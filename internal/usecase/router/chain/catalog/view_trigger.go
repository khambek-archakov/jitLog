package catalog

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackViewPrefix = "catalog:view:"

// viewTrigger opens a single catalog entry's card (catalog:view:{id}).
type viewTrigger struct {
	info catalogInfo
}

func (h *viewTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackViewPrefix) {
		return model.ErrSkip
	}

	return h.info.Handle(ctx, u, in)
}
