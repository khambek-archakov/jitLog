// Package list owns the "📅 Расписание" screen — the recurring weekly
// slots a user has set, plus the entry point into schedule/create. Tapping
// a slot hands off to schedule/info via its own schedule:view:{id}
// callback, exactly how training/history hands off to training/info.
package list

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	callbackScheduleList = "schedule:list"
	callbackScheduleAdd  = "schedule:add"
)

// callbackMenuBack mirrors other scenarios' own private constant.
const callbackMenuBack = "menu:back"

type UseCase struct {
	bot  sender
	repo slotRepo
}

func New(bot sender, repo slotRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

func (uc *UseCase) Handle(ctx context.Context, userID int64, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	if in.CallbackData != callbackScheduleList {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	slots, err := uc.repo.ListSlots(ctx, userID)
	if err != nil {
		return fmt.Errorf("list schedule slots: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, listText(slots), listKeyboard(slots))
}
