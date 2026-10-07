package competition

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackHistoryPagePrefix = "competition:history:page:"

// historyTrigger covers the paginated past-tournaments screen
// (competition:history:page:{n}), reached from the list screen's own
// "Прошедшие" button.
type historyTrigger struct {
	history competitionHistory
}

func (h *historyTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackHistoryPagePrefix) {
		return model.ErrSkip
	}

	return h.history.Handle(ctx, u, in)
}
