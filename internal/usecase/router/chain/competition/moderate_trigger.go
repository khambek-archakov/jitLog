package competition

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackModeratePrefix = "competition:moderate:"

// moderateTrigger covers every competition:moderate:* callback — the
// admin's ✅/❌/🔗 buttons and the merge picker results.
type moderateTrigger struct {
	moderate competitionModerate
}

func (h *moderateTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackModeratePrefix) {
		return model.ErrSkip
	}

	return h.moderate.Handle(ctx, u, in)
}
