package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// editDraftContinuation mirrors draftContinuation for update's own pending
// free-text edit (duration's "Другое" or notes).
type editDraftContinuation struct {
	update trainingUpdate
	edits  trainingEditDraft
}

func (h *editDraftContinuation) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	draft, err := h.edits.GetEditDraftByUserID(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		return ErrSkip
	}
	if err != nil {
		return fmt.Errorf("get training edit draft: %w", err)
	}

	return h.update.Continue(ctx, draft, in)
}
