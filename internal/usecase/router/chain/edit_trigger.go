package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackTrainingEditPrefix = "training:edit:"

// editTrigger covers every training:edit:* callback — the menu, each
// field's picker, and the stateless quick-value picks.
type editTrigger struct {
	update trainingUpdate
}

func (h *editTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackTrainingEditPrefix) {
		return ErrSkip
	}

	return h.update.Handle(ctx, u.ID, in)
}
