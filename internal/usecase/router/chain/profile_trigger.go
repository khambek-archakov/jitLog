package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackProfilePrefix = "profile:"

// profileTrigger covers every profile:* callback — the main menu's own
// "👤 Профиль" button (which points at profile:show) and the belt-change
// flow underneath it.
type profileTrigger struct {
	profile profile
}

func (h *profileTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackProfilePrefix) {
		return model.ErrSkip
	}

	return h.profile.Handle(ctx, u, in)
}
