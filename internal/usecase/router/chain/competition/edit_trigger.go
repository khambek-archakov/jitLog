package competition

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackEditPrefix = "competition:edit:"

// editTrigger covers every competition:edit:* callback — the menu and
// each field's free-text prompt.
type editTrigger struct {
	update competitionUpdate
}

func (h *editTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackEditPrefix) {
		return model.ErrSkip
	}

	return h.update.Handle(ctx, u, in)
}
