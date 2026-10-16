package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackWinbackPrefix = "winback:"

// winbackTrigger covers the two buttons on whatever standalone winback
// message internal/usecase/winback's own ticker sent — the only
// winback:*-prefixed callback that ever reaches the bot.
type winbackTrigger struct {
	winback winback
}

func (h *winbackTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackWinbackPrefix) {
		return model.ErrSkip
	}

	return h.winback.Handle(ctx, u, in)
}
