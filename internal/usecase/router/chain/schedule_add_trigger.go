package chain

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackScheduleAdd mirrors the schedule list screen's own private
// constant — the main-menu-adjacent "➕ Добавить" button.
const callbackScheduleAdd = "schedule:add"

// scheduleAddTrigger starts a fresh schedule-slot dialog from the
// "📅 Расписание" screen's own "➕ Добавить" button.
type scheduleAddTrigger struct {
	create scheduleCreate
}

func (h *scheduleAddTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || in.CallbackData != callbackScheduleAdd {
		return ErrSkip
	}

	return h.create.Begin(ctx, u.ID, in)
}
