package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackScheduleDeletePrefix = "schedule:delete:"

// scheduleDeleteTrigger covers schedule:delete:* — the confirm screen and
// the actual delete.
type scheduleDeleteTrigger struct {
	delete scheduleDelete
}

func (h *scheduleDeleteTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackScheduleDeletePrefix) {
		return ErrSkip
	}

	return h.delete.Handle(ctx, u.ID, in)
}
