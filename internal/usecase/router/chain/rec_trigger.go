package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackRecPrefix = "rec:"

// recTrigger covers the "✅ Записать" button on the post-session reminder
// message (see internal/usecase/reminder) — a separate top-level prefix
// from reminder:*, dispatched to the exact same underlying usecase, which
// already knows how to tell the two apart.
type recTrigger struct {
	reminder reminderDisable
}

func (h *recTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackRecPrefix) {
		return model.ErrSkip
	}

	return h.reminder.Handle(ctx, u, in)
}
