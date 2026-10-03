package schedule

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackList mirrors the schedule list screen's own private constant —
// the main menu's own button points at it.
const callbackList = "schedule:list"

// listTrigger covers the "📅 Расписание" screen's own entry callback.
// Editing/deleting a specific slot goes through editTrigger/deleteTrigger
// instead, both reached before this link — see New's ordering.
type listTrigger struct {
	list scheduleList
}

func (h *listTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || in.CallbackData != callbackList {
		return model.ErrSkip
	}

	return h.list.Handle(ctx, u.ID, in)
}
