package schedule

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackAdd mirrors the schedule list screen's own private constant —
// the "➕ Добавить" button.
const callbackAdd = "schedule:add"

// addTrigger starts a fresh schedule-slot dialog from the
// "📅 Расписание" screen's own "➕ Добавить" button.
type addTrigger struct {
	create scheduleCreate
}

func (h *addTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || in.CallbackData != callbackAdd {
		return model.ErrSkip
	}

	return h.create.Begin(ctx, u.ID, in)
}
