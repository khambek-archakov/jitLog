package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackStatsPeriodPrefix = "stats:period:"

// statsTrigger covers every stats:period:* callback — the main menu's own
// "📊 Статистика" button and every period-tab on the stats screen itself.
type statsTrigger struct {
	stats trainingStats
}

func (h *statsTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackStatsPeriodPrefix) {
		return ErrSkip
	}

	return h.stats.Handle(ctx, u.ID, in)
}
