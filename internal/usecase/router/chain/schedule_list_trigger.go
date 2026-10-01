package chain

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackScheduleList mirrors the schedule list screen's own private
// constant — the main menu's own button points at it.
const callbackScheduleList = "schedule:list"

// scheduleListTrigger covers the "📅 Расписание" screen's own entry
// callback. Editing/deleting a specific slot goes through
// scheduleEditTrigger/scheduleDeleteTrigger instead, both reached before
// this link — see Default's ordering.
type scheduleListTrigger struct {
	list scheduleList
}

func (h *scheduleListTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || in.CallbackData != callbackScheduleList {
		return ErrSkip
	}

	return h.list.Handle(ctx, u.ID, in)
}
