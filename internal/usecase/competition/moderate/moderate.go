// Package moderate owns the admin's three moderation actions on a pending
// catalog submission (✅ Одобрить / ❌ Отклонить / 🔗 Это дубль) plus the
// free-text title search "🔗 Это дубль" starts (Continue). Every action
// here is guarded against Telegram redelivering the same callback twice —
// see repo.SetCompetitionStatus and repo.MergeCompetition's own
// status/existence guards — by answering "already handled" instead of
// erroring or double-notifying.
package moderate

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	alreadyHandledText = "Уже обработано."
	noAccessText       = "Нет доступа."
	noMorePendingText  = "Больше нет соревнований на проверке."
)

// callbackModeratePrefix is followed by "{id}:approve", "{id}:reject",
// "{id}:dup" or "{id}:merge:{existingID}" for a single competition, or by
// one of the three fixed "ajp:..." actions below for an automated
// import's own batch/diff flow.
const callbackModeratePrefix = "competition:moderate:"

// callbackApproveAll and callbackReviewPrefix back an automated import's
// "✅ Одобрить все" / "👀 Смотреть по одному" — shown instead of per-item
// buttons on the first-ever import run or whenever one run creates more
// than 10 pending competitions. Both act on every pending competition
// without an author (created_by is null — only an automated import ever
// leaves that), live at the moment they're tapped, not just the batch one
// particular run produced.
const (
	callbackApproveAll   = "competition:moderate:ajp:approve_all"
	callbackReviewPrefix = "competition:moderate:ajp:review:"
)

// callbackDiffPrefix is followed by "{id}:apply" or "{id}:keep" — an
// automated import's own "✏️ Обновить" / "➡️ Оставить" on a detected
// date/city drift (see competition_import_diff).
const callbackDiffPrefix = "competition:moderate:ajp:diff:"

// callbackCompetitionList mirrors internal/usecase/competition/list's own
// private constant — every owner notification below ends with a button
// into that screen, since that's where the now-updated status is visible.
const callbackCompetitionList = "competition:list"

type UseCase struct {
	bot             sender
	repo            repo
	users           userRepo
	adminTelegramID int64
}

func New(bot sender, r repo, users userRepo, adminTelegramID int64) *UseCase {
	return &UseCase{bot: bot, repo: r, users: users, adminTelegramID: adminTelegramID}
}

// Handle dispatches every competition:moderate:* callback — Router only
// reaches this usecase for that prefix. The admin check here is defense
// in depth: in practice nobody else's chat ever receives these buttons to
// tap in the first place.
func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	if u.TelegramID != uc.adminTelegramID {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, noAccessText)
	}

	switch {
	case in.CallbackData == callbackApproveAll:
		return uc.approveAllPendingUnauthored(ctx, in)

	case strings.HasPrefix(in.CallbackData, callbackReviewPrefix):
		return uc.reviewPendingUnauthored(ctx, in)

	case strings.HasPrefix(in.CallbackData, callbackDiffPrefix):
		return uc.handleDiff(ctx, in)
	}

	rest := strings.TrimPrefix(in.CallbackData, callbackModeratePrefix)

	idStr, actionRest, _ := strings.Cut(rest, ":")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	action, arg, _ := strings.Cut(actionRest, ":")

	switch action {
	case "approve":
		return uc.setStatus(ctx, in, id, model.CompetitionStatusPublished, "✅ Соревнование опубликовано в каталоге!", "✅ Одобрено.")

	case "reject":
		return uc.setStatus(ctx, in, id, model.CompetitionStatusRejected, "❌ Соревнование отклонено модератором.", "❌ Отклонено.")

	case "dup":
		return uc.beginDup(ctx, u, in, id)

	case "merge":
		existingID, err := strconv.ParseInt(arg, 10, 64)
		if err != nil {
			return uc.bot.AnswerCallback(ctx, in.CallbackID)
		}

		return uc.merge(ctx, u, in, id, existingID)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

// Continue applies a free-text title search against the pending merge
// draft — "🔗 Это дубль"'s own free-text step. It deliberately does not
// delete the draft on a search (even a zero-result one) — only an actual
// merge does — so the admin can retype and search again.
func (uc *UseCase) Continue(ctx context.Context, u *model.User, d *model.CompetitionMergeDraft, in dto.Input) error {
	if !in.HasMessage {
		return nil
	}

	term := strings.TrimSpace(in.Text)
	if term == "" {
		return uc.bot.Send(ctx, in.ChatID, "Напиши часть названия соревнования.")
	}

	results, err := uc.repo.SearchCompetitionsByTitle(ctx, term, d.PendingCompetitionID, 10)
	if err != nil {
		return fmt.Errorf("search competitions by title: %w", err)
	}

	if len(results) == 0 {
		return uc.bot.Send(ctx, in.ChatID, "Ничего не нашёл, попробуй другую часть названия.")
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, "Это один из них?", mergeCandidateKeyboard(d.PendingCompetitionID, results))
}

func (uc *UseCase) setStatus(
	ctx context.Context, in dto.Input, id int64, status model.CompetitionStatus, notifyText, doneText string,
) error {
	changed, err := uc.repo.SetCompetitionStatus(ctx, id, status)
	if err != nil {
		return fmt.Errorf("set competition status: %w", err)
	}

	if !changed {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, alreadyHandledText)
	}

	if err := uc.notifyOwners(ctx, id, notifyText); err != nil {
		return err
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, doneText, nil)
}

func (uc *UseCase) beginDup(ctx context.Context, u *model.User, in dto.Input, pendingID int64) error {
	if err := uc.repo.SetMergeDraft(ctx, u.ID, pendingID); err != nil {
		return fmt.Errorf("set competition merge draft: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.Send(ctx, in.ChatID, "Напиши часть названия соревнования, с которым нужно объединить.")
}

func (uc *UseCase) merge(ctx context.Context, u *model.User, in dto.Input, pendingID, existingID int64) error {
	owners, err := uc.repo.ListOwnersByCompetitionID(ctx, pendingID)
	if err != nil {
		return fmt.Errorf("list competition owners: %w", err)
	}

	merged, err := uc.repo.MergeCompetition(ctx, pendingID, existingID)
	if err != nil {
		return fmt.Errorf("merge competition: %w", err)
	}

	if !merged {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, alreadyHandledText)
	}

	if err := uc.repo.DeleteMergeDraft(ctx, u.ID); err != nil {
		return fmt.Errorf("delete competition merge draft: %w", err)
	}

	if err := uc.notifyOwnersDirect(ctx, owners, "Это соревнование уже есть в каталоге, я добавил его к существующему."); err != nil {
		return err
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "🔗 Объединено с существующим соревнованием.", nil)
}

func (uc *UseCase) notifyOwners(ctx context.Context, competitionID int64, text string) error {
	owners, err := uc.repo.ListOwnersByCompetitionID(ctx, competitionID)
	if err != nil {
		return fmt.Errorf("list competition owners: %w", err)
	}

	return uc.notifyOwnersDirect(ctx, owners, text)
}

func (uc *UseCase) notifyOwnersDirect(ctx context.Context, ownerIDs []int64, text string) error {
	for _, ownerID := range ownerIDs {
		owner, err := uc.users.GetByID(ctx, ownerID)
		if err != nil {
			return fmt.Errorf("get owner: %w", err)
		}

		if err := uc.bot.SendWithKeyboard(ctx, owner.TelegramID, text, competitionListKeyboard()); err != nil {
			return err
		}
	}

	return nil
}

func competitionListKeyboard() dto.Keyboard {
	return dto.Keyboard{dto.Row(dto.Button{Label: "🏆 Соревнования", Data: callbackCompetitionList})}
}

// approveAllPendingUnauthored publishes every pending, author-less
// competition in one go — owners are never notified here, since nobody's
// own UserCompetition links to an import-created entry until someone
// taps "➕ В мои" on it later.
func (uc *UseCase) approveAllPendingUnauthored(ctx context.Context, in dto.Input) error {
	pending, err := uc.repo.ListPendingUnauthored(ctx)
	if err != nil {
		return fmt.Errorf("list pending unauthored competitions: %w", err)
	}

	var approved int

	for _, c := range pending {
		changed, err := uc.repo.SetCompetitionStatus(ctx, c.ID, model.CompetitionStatusPublished)
		if err != nil {
			return fmt.Errorf("set competition status: %w", err)
		}

		if changed {
			approved++
		}
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, fmt.Sprintf("✅ Одобрено %d соревнований.", approved), nil,
	)
}

// reviewPendingUnauthored shows one pending, author-less competition at a
// time — index is a 0-based position into the live (not a frozen-at-tap)
// list, so acting on the current one and then tapping "➡️ Следующее"
// always surfaces whatever is now first in line. It sends a fresh message
// rather than editing the tapped one, so the original "✅ Одобрить все" /
// "👀 Смотреть по одному" summary stays reachable throughout.
func (uc *UseCase) reviewPendingUnauthored(ctx context.Context, in dto.Input) error {
	index, err := strconv.Atoi(strings.TrimPrefix(in.CallbackData, callbackReviewPrefix))
	if err != nil || index < 0 {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	pending, err := uc.repo.ListPendingUnauthored(ctx)
	if err != nil {
		return fmt.Errorf("list pending unauthored competitions: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	if index >= len(pending) {
		return uc.bot.Send(ctx, in.ChatID, noMorePendingText)
	}

	c := pending[index]

	url, err := uc.repo.GetPrimarySourceURL(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get primary competition source: %w", err)
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, reviewText(c, url), reviewKeyboard(c.ID, index))
}

func reviewText(c *model.Competition, url *string) string {
	link := "—"
	if url != nil {
		link = *url
	}

	city := "—"
	if c.City != nil && *c.City != "" {
		city = *c.City
	}

	return fmt.Sprintf("📥 На проверке\n\n%s\n📅 %s\n🏙 %s\n🔗 %s", c.Title, info.FormatDate(c.Date), city, link)
}

func reviewKeyboard(id int64, index int) dto.Keyboard {
	prefix := fmt.Sprintf("%s%d:", callbackModeratePrefix, id)

	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "✅ Одобрить", Data: prefix + "approve"},
			dto.Button{Label: "❌ Отклонить", Data: prefix + "reject"},
		),
		dto.Row(dto.Button{Label: "🔗 Это дубль", Data: prefix + "dup"}),
		dto.Row(dto.Button{Label: "➡️ Следующее", Data: fmt.Sprintf("%s%d", callbackReviewPrefix, index+1)}),
	}
}

// handleDiff applies or dismisses an automated import's proposed
// date/city change for an already-known competition — see
// model.CompetitionImportDiff.
func (uc *UseCase) handleDiff(ctx context.Context, in dto.Input) error {
	rest := strings.TrimPrefix(in.CallbackData, callbackDiffPrefix)

	idStr, action, _ := strings.Cut(rest, ":")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	switch action {
	case "apply":
		applied, err := uc.repo.ApplyImportDiff(ctx, id)
		if err != nil {
			return fmt.Errorf("apply competition import diff: %w", err)
		}

		if !applied {
			return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, alreadyHandledText)
		}

		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "✅ Обновлено.", nil)

	case "keep":
		if err := uc.repo.DeleteImportDiff(ctx, id); err != nil {
			return fmt.Errorf("delete competition import diff: %w", err)
		}

		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "Оставлено как есть.", nil)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}
