package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackScheduleEditPrefix = "schedule:edit:"

// scheduleEditTrigger covers every schedule:edit:* callback — the menu,
// each field's picker, and the stateless quick-value picks.
type scheduleEditTrigger struct {
	update scheduleUpdate
}

func (h *scheduleEditTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackScheduleEditPrefix) {
		return ErrSkip
	}

	return h.update.Handle(ctx, u.ID, in)
}
