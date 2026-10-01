package chain

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackMenuAddTraining mirrors the main menu's own button data — the
// only trigger in this package that isn't a callback-data prefix.
const callbackMenuAddTraining = "menu:add_training"

// addTrainingTrigger starts a fresh create dialog from the main menu's
// button.
type addTrainingTrigger struct {
	create trainingCreate
}

func (h *addTrainingTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || in.CallbackData != callbackMenuAddTraining {
		return ErrSkip
	}

	return h.create.Begin(ctx, u.ID, in)
}
