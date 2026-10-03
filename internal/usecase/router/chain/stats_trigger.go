package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackStatsPrefix covers both of stats' own callback namespaces:
// stats:period:* (the main menu's own button, and every period tab) and
// stats:belts (the "По поясам" mode switch).
const callbackStatsPrefix = "stats:"

// statsTrigger covers every stats:* callback — the main menu's own
// "📊 Статистика" button and everything on the stats screen itself.
type statsTrigger struct {
	stats trainingStats
}

func (h *statsTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackStatsPrefix) {
		return model.ErrSkip
	}

	return h.stats.Handle(ctx, u.ID, in)
}
