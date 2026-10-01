package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackTrainingViewPrefix = "training:view:"

// viewTrigger opens a single training's card (training:view:{id}).
type viewTrigger struct {
	info trainingInfo
}

func (h *viewTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackTrainingViewPrefix) {
		return ErrSkip
	}

	return h.info.Handle(ctx, u.ID, in)
}
