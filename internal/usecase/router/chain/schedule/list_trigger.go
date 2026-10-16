package schedule

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackList mirrors the schedule list screen's own private constant —
// the main menu's own button points at it.
const callbackList = "schedule:list"

// callbackReminderPrefix/callbackTZPrefix mirror the schedule list
// screen's own private constants — the reminder toggle and its timezone
// picker both live on this same screen.
const (
	callbackReminderPrefix = "schedule:reminders:"
	callbackTZPrefix       = "schedule:tz:"
)

// listTrigger covers the "Расписание" screen's own entry callback, plus
// its reminder toggle and timezone picker. Editing/deleting a specific
// slot goes through editTrigger/deleteTrigger instead, both reached before
// this link — see New's ordering.
type listTrigger struct {
	list scheduleList
}

func (h *listTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return model.ErrSkip
	}

	matches := in.CallbackData == callbackList ||
		strings.HasPrefix(in.CallbackData, callbackReminderPrefix) ||
		strings.HasPrefix(in.CallbackData, callbackTZPrefix)
	if !matches {
		return model.ErrSkip
	}

	return h.list.Handle(ctx, u, in)
}
