// Package delete owns removing an already-logged training: the
// training:delete:{id} entry point (a confirm screen) and
// training:delete:confirm:{id} (the actual delete). Both are fully
// stateless — the id rides the callback data the whole way, same as most
// of internal/usecase/training/update.
package delete

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/info"
)

const notFoundText = "Тренировка не найдена."

const (
	callbackDeletePrefix        = "training:delete:"
	callbackDeleteConfirmPrefix = "training:delete:confirm:"
	// callbackHistoryFirstPage mirrors history's own first-page callback —
	// after a successful delete there's nothing left to show a card for.
	callbackHistoryFirstPage = "training:history:page:0"
)

type UseCase struct {
	bot  sender
	repo trainingRepo
}

func New(bot sender, repo trainingRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

// Handle dispatches every training:delete:* callback — Router only reaches
// this usecase for that prefix.
func (uc *UseCase) Handle(ctx context.Context, userID int64, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	if strings.HasPrefix(in.CallbackData, callbackDeleteConfirmPrefix) {
		return uc.confirm(ctx, userID, in)
	}

	id, ok := parseID(in.CallbackData, callbackDeletePrefix)
	if !ok {
		return uc.bot.AnswerCallback(in.CallbackID)
	}

	t, err := uc.getOwnTraining(ctx, id, userID)
	if err != nil {
		return err
	}
	if t == nil {
		return uc.bot.AnswerCallbackWithText(in.CallbackID, notFoundText)
	}

	if err := uc.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(in.ChatID, in.MessageID, confirmText(t), confirmKeyboard(t.ID))
}

func (uc *UseCase) confirm(ctx context.Context, userID int64, in dto.Input) error {
	id, ok := parseID(in.CallbackData, callbackDeleteConfirmPrefix)
	if !ok {
		return uc.bot.AnswerCallback(in.CallbackID)
	}

	t, err := uc.getOwnTraining(ctx, id, userID)
	if err != nil {
		return err
	}
	if t == nil {
		return uc.bot.AnswerCallbackWithText(in.CallbackID, notFoundText)
	}

	if err := uc.repo.DeleteTraining(ctx, id); err != nil {
		return fmt.Errorf("delete training: %w", err)
	}

	if err := uc.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(in.ChatID, in.MessageID, "🗑 Тренировка удалена.", deletedKeyboard())
}

// getOwnTraining fetches a training and checks it belongs to userID,
// returning (nil, nil) for "not found or not yours" so callers can send
// the same notFoundText either way without leaking which case it was.
func (uc *UseCase) getOwnTraining(ctx context.Context, id, userID int64) (*model.Training, error) {
	t, err := uc.repo.GetTraining(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get training: %w", err)
	}

	if t.UserID != userID {
		return nil, nil
	}

	return t, nil
}

func confirmText(t *model.Training) string {
	return "🗑 Удалить тренировку?\n\n" + info.Body(t)
}

func confirmKeyboard(id int64) dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "Да, удалить", Data: fmt.Sprintf("%s%d", callbackDeleteConfirmPrefix, id)}),
		dto.Row(dto.Button{Label: "Отмена", Data: fmt.Sprintf("training:view:%d", id)}),
	}
}

func deletedKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "← Мои тренировки", Data: callbackHistoryFirstPage}),
	}
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
