package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// profileEditDraftContinuation mirrors scheduleEditDraftContinuation for
// profile's own pending free-text edit (age). It only ever claims a
// message — a callback (the prompt's own "❌ Отмена" button) is left for
// the profile:* trigger to dispatch to Handle instead, since Continue only
// understands free text.
type profileEditDraftContinuation struct {
	profile profile
	edits   profileEditDraft
}

func (h *profileEditDraftContinuation) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if in.HasCallback {
		return model.ErrSkip
	}

	draft, err := h.edits.GetEditDraftByUserID(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		return model.ErrSkip
	}
	if err != nil {
		return fmt.Errorf("get profile edit draft: %w", err)
	}

	return h.profile.Continue(ctx, u, draft, in)
}
