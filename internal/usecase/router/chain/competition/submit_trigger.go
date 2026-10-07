package competition

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackSubmitPrefix = "competition:submit:"

// submitTrigger covers every competition:submit:* callback — the initial
// tap, candidate picks and the "none of these" bypass.
type submitTrigger struct {
	submit competitionSubmit
}

func (h *submitTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackSubmitPrefix) {
		return model.ErrSkip
	}

	return h.submit.Handle(ctx, u, in)
}
