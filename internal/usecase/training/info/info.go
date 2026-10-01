// Package info owns the single-training card: the training:view:{id}
// entry point (tapping a row in the history list), plus the card-body
// formatting that create's "saved" screen and update's return-to-card
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

const notFoundText = "Тренировка не найдена."

type UseCase struct {
	bot  sender
	repo trainingRepo
}

func New(bot sender, repo trainingRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

// Handle shows the card for a training:view:{id} callback. Router only
// reaches this usecase for callbacks carrying that prefix, so anything
// else here is just a malformed id.
func (uc *UseCase) Handle(ctx context.Context, userID int64, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	id, ok := parseTrainingID(in.CallbackData, callbackTrainingViewPrefix)
	if !ok {
		return uc.bot.AnswerCallback(in.CallbackID)
	}

	t, err := uc.repo.GetTraining(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return uc.bot.AnswerCallbackWithText(in.CallbackID, notFoundText)
	}
	if err != nil {
		return fmt.Errorf("get training: %w", err)
	}

	if t.UserID != userID {
		return uc.bot.AnswerCallbackWithText(in.CallbackID, notFoundText)
	}

	if err := uc.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(in.ChatID, in.MessageID, Card(t), Keyboard(t.ID))
}

func parseTrainingID(data, prefix string) (int64, bool) {
	if !strings.HasPrefix(data, prefix) {
		return 0, false
	}

	id, err := strconv.ParseInt(strings.TrimPrefix(data, prefix), 10, 64)
	if err != nil {
		return 0, false
	}

	return id, true
}
