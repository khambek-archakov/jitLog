// Package delete owns removing a schedule slot: the schedule:delete:{id}
// entry point (a confirm screen) and schedule:delete:confirm:{id} (the
// actual delete). Both are fully stateless — the id rides the callback
// data the whole way, same as training/delete.
package delete

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/info"
)

const notFoundText = "Слот не найден."

const (
	callbackDeletePrefix        = "schedule:delete:"
	callbackDeleteConfirmPrefix = "schedule:delete:confirm:"
	// callbackScheduleList mirrors internal/usecase/schedule/list's own
	// private constant — after a successful delete there's nothing left
	// to show a card for.
	callbackScheduleList = "schedule:list"
)

type UseCase struct {
	bot  sender
	repo slotRepo
}

func New(bot sender, repo slotRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

// Handle dispatches every schedule:delete:* callback — Router only reaches
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
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	s, err := uc.getOwnSlot(ctx, id, userID)
	if err != nil {
		return err
	}
	if s == nil {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, confirmText(s), confirmKeyboard(s.ID))
}

func (uc *UseCase) confirm(ctx context.Context, userID int64, in dto.Input) error {
	id, ok := parseID(in.CallbackData, callbackDeleteConfirmPrefix)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	s, err := uc.getOwnSlot(ctx, id, userID)
	if err != nil {
		return err
	}
	if s == nil {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}

	if err := uc.repo.DeleteSlot(ctx, id); err != nil {
		return fmt.Errorf("delete schedule slot: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "🗑 Слот удалён.", deletedKeyboard())
}

// getOwnSlot fetches a slot and checks it belongs to userID, returning
// (nil, nil) for "not found or not yours" so callers can send the same
// notFoundText either way without leaking which case it was.
func (uc *UseCase) getOwnSlot(ctx context.Context, id, userID int64) (*model.ScheduleSlot, error) {
	s, err := uc.repo.GetSlot(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get schedule slot: %w", err)
	}

	if s.UserID != userID {
		return nil, nil
	}

	return s, nil
}

func confirmText(s *model.ScheduleSlot) string {
	return "🗑 Удалить слот?\n\n" + info.Body(s)
}

func confirmKeyboard(id int64) dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "Да, удалить", Data: fmt.Sprintf("%s%d", callbackDeleteConfirmPrefix, id)}),
		dto.Row(dto.Button{Label: "Отмена", Data: fmt.Sprintf("schedule:view:%d", id)}),
	}
}

func deletedKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "← Расписание", Data: callbackScheduleList}),
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
