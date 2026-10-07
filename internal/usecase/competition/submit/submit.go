// Package submit owns "📤 Предложить в каталог" — turning a personal
// tournament entry (which must already have a URL) into a shared catalog
// one, via a three-step dedup chain: an exact source-URL match attaches
// silently, a date±2-days/same-city candidate lets the submitter confirm
// it by hand, and only when neither applies does a brand new pending
// catalog entry get created and sent to the admin for moderation (see
// sibling package competition/moderate).
package submit

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	notFoundText = "Соревнование не найдено."
	needsURLText = "Нужна ссылка, чтобы предложить соревнование в каталог."
)

// callbackSubmitPrefix is followed by "{id}" (the initial tap),
// "{id}:attach:{candidateID}" (picking a candidate) or "{id}:new"
// (declining every candidate).
const callbackSubmitPrefix = "competition:submit:"

type UseCase struct {
	bot         sender
	repo        repo
	adminChatID int64
}

func New(bot sender, r repo, adminChatID int64) *UseCase {
	return &UseCase{bot: bot, repo: r, adminChatID: adminChatID}
}

// Handle dispatches every competition:submit:* callback — Router only
// reaches this usecase for that prefix.
func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	rest := strings.TrimPrefix(in.CallbackData, callbackSubmitPrefix)

	idStr, action, _ := strings.Cut(rest, ":")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	c, err := uc.repo.GetUserCompetition(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}
	if err != nil {
		return fmt.Errorf("get user competition: %w", err)
	}

	if c.UserID != u.ID {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}

	if c.URL == nil || *c.URL == "" {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, needsURLText)
	}

	switch {
	case action == "":
		return uc.begin(ctx, u, c, in)

	case action == "new":
		return uc.createNew(ctx, u, c, in)

	case strings.HasPrefix(action, "attach:"):
		candidateID, err := strconv.ParseInt(strings.TrimPrefix(action, "attach:"), 10, 64)
		if err != nil {
			return uc.bot.AnswerCallback(ctx, in.CallbackID)
		}

		return uc.attach(ctx, c, in, candidateID)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

// begin runs the dedup chain: exact URL match, then a date/city candidate
// search (only possible if the record has a city at all), then falls
// through to creating a brand new catalog entry.
func (uc *UseCase) begin(ctx context.Context, u *model.User, c *model.UserCompetition, in dto.Input) error {
	source, err := uc.repo.FindSourceByURL(ctx, *c.URL)
	if err == nil {
		if err := uc.repo.LinkUserCompetition(ctx, c.ID, source.CompetitionID); err != nil {
			return fmt.Errorf("link user competition: %w", err)
		}

		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.Send(ctx, in.ChatID, "Такое соревнование уже есть в каталоге — привязал твою запись к нему.")
	}
	if !errors.Is(err, model.ErrNotFound) {
		return fmt.Errorf("find competition source by url: %w", err)
	}

	if c.City != nil && *c.City != "" {
		candidates, err := uc.repo.FindCandidates(ctx, c.Date, *c.City)
		if err != nil {
			return fmt.Errorf("find candidate competitions: %w", err)
		}

		if len(candidates) > 0 {
			if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
				return err
			}

			return uc.bot.SendWithKeyboard(ctx, in.ChatID, "Это один из них?", candidateKeyboard(c.ID, candidates))
		}
	}

	return uc.createNew(ctx, u, c, in)
}

func (uc *UseCase) attach(ctx context.Context, c *model.UserCompetition, in dto.Input, candidateID int64) error {
	if err := uc.repo.AddCompetitionSource(ctx, candidateID, *c.URL, false); err != nil {
		return fmt.Errorf("add competition source: %w", err)
	}

	if err := uc.repo.LinkUserCompetition(ctx, c.ID, candidateID); err != nil {
		return fmt.Errorf("link user competition: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.Send(ctx, in.ChatID, "Привязал к существующему соревнованию в каталоге.")
}

func (uc *UseCase) createNew(ctx context.Context, u *model.User, c *model.UserCompetition, in dto.Input) error {
	created, err := uc.repo.CreateCompetition(ctx, c.Title, c.Date, c.EndDate, c.City, u.ID)
	if err != nil {
		return fmt.Errorf("create competition: %w", err)
	}

	if err := uc.repo.AddCompetitionSource(ctx, created.ID, *c.URL, true); err != nil {
		return fmt.Errorf("add competition source: %w", err)
	}

	if err := uc.repo.LinkUserCompetition(ctx, c.ID, created.ID); err != nil {
		return fmt.Errorf("link user competition: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	if err := uc.bot.Send(ctx, in.ChatID, "Отправил соревнование на модерацию — админ скоро проверит."); err != nil {
		return err
	}

	return uc.bot.SendWithKeyboard(ctx, uc.adminChatID, moderationText(created, *c.URL), moderationKeyboard(created.ID))
}
