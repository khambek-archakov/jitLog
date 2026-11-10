package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackSharePrefix covers the "↗️ Поделиться" monthly recap image (see
// internal/usecase/training/stats) — a separate top-level prefix from
// stats:*, dispatched to the exact same underlying usecase, which
// already knows how to tell it apart from every other stats screen.
const callbackSharePrefix = "share:"

type shareTrigger struct {
	stats trainingStats
}

func (h *shareTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackSharePrefix) {
		return model.ErrSkip
	}

	return h.stats.Handle(ctx, u, in)
}
