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
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	alreadyHandledText = "Уже обработано."
	noAccessText       = "Нет доступа."
)

// callbackModeratePrefix is followed by "{id}:approve", "{id}:reject",
// "{id}:dup" or "{id}:merge:{existingID}".
const callbackModeratePrefix = "competition:moderate:"

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

	rest := strings.TrimPrefix(in.CallbackData, callbackModeratePrefix)

	idStr, actionRest, _ := strings.Cut(rest, ":")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	action, arg, _ := strings.Cut(actionRest, ":")

	switch action {
	case "approve":
		return uc.setStatus(ctx, in, id, model.CompetitionStatusPublished, "✅ Турнир опубликован в каталоге!", "✅ Одобрено.")

	case "reject":
		return uc.setStatus(ctx, in, id, model.CompetitionStatusRejected, "❌ Турнир отклонён модератором.", "❌ Отклонено.")

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
		return uc.bot.Send(ctx, in.ChatID, "Напиши часть названия турнира.")
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

	return uc.bot.Send(ctx, in.ChatID, "Напиши часть названия турнира, с которым нужно объединить.")
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

	if err := uc.notifyOwnersDirect(ctx, owners, "Этот турнир уже есть в каталоге, я добавил его к существующему."); err != nil {
		return err
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "🔗 Объединено с существующим турниром.", nil)
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

		if err := uc.bot.Send(ctx, owner.TelegramID, text); err != nil {
			return err
		}
	}

	return nil
}
