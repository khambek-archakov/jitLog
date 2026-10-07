package competition

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackList mirrors the competition list screen's own private constant —
// the main menu's own button points at it.
const callbackList = "competition:list"

// listTrigger covers the "🏆 Соревнования" screen's own entry callback.
// Viewing/editing/deleting a specific tournament goes through
// viewTrigger/editTrigger/deleteTrigger instead, all reached before this
// link — see New's ordering.
type listTrigger struct {
	list competitionList
}

func (h *listTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || in.CallbackData != callbackList {
		return model.ErrSkip
	}

	return h.list.Handle(ctx, u, in)
}
