package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackCalPrefix covers the "📅 Другие даты" custom-range calendar
// (see internal/usecase/training/stats) — a separate top-level prefix
// from stats:*, dispatched to the exact same underlying usecase, which
// already knows how to tell the two apart.
const callbackCalPrefix = "cal:"

type calTrigger struct {
	stats trainingStats
}

func (h *calTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackCalPrefix) {
		return model.ErrSkip
	}

	return h.stats.Handle(ctx, u, in)
}
