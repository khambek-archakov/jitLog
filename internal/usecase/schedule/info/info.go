// Package info owns the single-slot card: the schedule:view:{id} entry
// point (tapping a row in the schedule list), plus the card-body
// formatting that create's confirmation screen and update's return-to-card
// screen also render.
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

const notFoundText = "Слот не найден."

const callbackViewPrefix = "schedule:view:"

type UseCase struct {
	bot  sender
	repo slotRepo
}

func New(bot sender, repo slotRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

// Handle shows the card for a schedule:view:{id} callback. Router only
// reaches this usecase for callbacks carrying that prefix, so anything
// else here is just a malformed id.
func (uc *UseCase) Handle(ctx context.Context, userID int64, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	id, ok := parseSlotID(in.CallbackData, callbackViewPrefix)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	s, err := uc.repo.GetSlot(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}
	if err != nil {
		return fmt.Errorf("get schedule slot: %w", err)
	}

	if s.UserID != userID {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, Card(s), Keyboard(s.ID))
}

func parseSlotID(data, prefix string) (int64, bool) {
	if !strings.HasPrefix(data, prefix) {
		return 0, false
	}

	id, err := strconv.ParseInt(strings.TrimPrefix(data, prefix), 10, 64)
	if err != nil {
		return 0, false
	}

	return id, true
}
