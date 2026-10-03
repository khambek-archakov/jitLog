package training

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackDeletePrefix = "training:delete:"

// deleteTrigger covers training:delete:* — the confirm screen and the
// actual delete.
type deleteTrigger struct {
	delete trainingDelete
}

func (h *deleteTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackDeletePrefix) {
		return model.ErrSkip
	}

	return h.delete.Handle(ctx, u.ID, in)
}
