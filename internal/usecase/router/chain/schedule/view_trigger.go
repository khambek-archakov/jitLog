package schedule

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackViewPrefix = "schedule:view:"

// viewTrigger opens a single slot's card (schedule:view:{id}).
type viewTrigger struct {
	info scheduleInfo
}

func (h *viewTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackViewPrefix) {
		return model.ErrSkip
	}

	return h.info.Handle(ctx, u.ID, in)
}
