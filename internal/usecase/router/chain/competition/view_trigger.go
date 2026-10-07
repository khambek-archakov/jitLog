package competition

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackViewPrefix = "competition:view:"

// viewTrigger opens a single tournament's card (competition:view:{id}).
type viewTrigger struct {
	info competitionInfo
}

func (h *viewTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackViewPrefix) {
		return model.ErrSkip
	}

	return h.info.Handle(ctx, u, in)
}
