package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackScheduleViewPrefix = "schedule:view:"

// scheduleViewTrigger opens a single slot's card (schedule:view:{id}).
type scheduleViewTrigger struct {
	info scheduleInfo
}

func (h *scheduleViewTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackScheduleViewPrefix) {
		return ErrSkip
	}

	return h.info.Handle(ctx, u.ID, in)
}
