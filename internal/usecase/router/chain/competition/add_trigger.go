package competition

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackAdd mirrors the competition list screen's own private constant —
// the "➕ Добавить" button.
const callbackAdd = "competition:add"

// addTrigger starts a fresh tournament dialog from the "🏆 Соревнования"
// screen's own "➕ Добавить" button.
type addTrigger struct {
	create competitionCreate
}

func (h *addTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || in.CallbackData != callbackAdd {
		return model.ErrSkip
	}

	return h.create.Begin(ctx, u, in)
}
