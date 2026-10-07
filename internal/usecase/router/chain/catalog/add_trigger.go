package catalog

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackAddPrefix = "catalog:add:"

// addTrigger owns "➕ В мои" (catalog:add:{id}).
type addTrigger struct {
	add catalogAdd
}

func (h *addTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackAddPrefix) {
		return model.ErrSkip
	}

	return h.add.Handle(ctx, u, in)
}
