// Package info owns the single-tournament card: the competition:view:{id}
// entry point (tapping a row in the upcoming/past lists), plus the
// card-body/status formatting that create's confirmation screen,
// update's return-to-card screen and delete's confirm screen also render.
package info

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const notFoundText = "Турнир не найден."

const callbackViewPrefix = "competition:view:"

type UseCase struct {
	bot  sender
	repo competitionRepo
}

func New(bot sender, repo competitionRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

// Handle shows the card for a competition:view:{id} callback. Router only
// reaches this usecase for callbacks carrying that prefix, so anything
// else here is just a malformed id.
func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	id, ok := parseID(in.CallbackData, callbackViewPrefix)
	if !ok {
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

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	today := Today(u)

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, Card(c, today), Keyboard(c, today))
}

func parseID(data, prefix string) (int64, bool) {
	if !strings.HasPrefix(data, prefix) {
		return 0, false
	}

	id, err := strconv.ParseInt(strings.TrimPrefix(data, prefix), 10, 64)
	if err != nil {
		return 0, false
	}

	return id, true
}
