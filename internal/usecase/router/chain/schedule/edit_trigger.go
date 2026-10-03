package schedule

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackEditPrefix = "schedule:edit:"

// editTrigger covers every schedule:edit:* callback — the menu, each
// field's picker, and the stateless quick-value picks.
type editTrigger struct {
	update scheduleUpdate
}

func (h *editTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackEditPrefix) {
		return model.ErrSkip
	}

	return h.update.Handle(ctx, u.ID, in)
}
