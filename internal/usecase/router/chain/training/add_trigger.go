package training

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackMenuAddTraining mirrors the main menu's own button data.
const callbackMenuAddTraining = "menu:add_training"

// addTrigger starts a fresh create dialog from the main menu's button.
type addTrigger struct {
	create trainingCreate
}

func (h *addTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || in.CallbackData != callbackMenuAddTraining {
		return model.ErrSkip
	}

	return h.create.Begin(ctx, u.ID, in)
}
