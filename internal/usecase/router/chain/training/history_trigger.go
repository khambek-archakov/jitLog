package training

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackHistoryPagePrefix = "training:history:page:"

// historyTrigger shows the paginated list (training:history:page:{n}).
type historyTrigger struct {
	history trainingHistory
}

func (h *historyTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackHistoryPagePrefix) {
		return model.ErrSkip
	}

	return h.history.Handle(ctx, u.ID, in)
}
