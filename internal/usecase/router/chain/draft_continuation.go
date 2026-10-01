package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// draftContinuation hands off to create.Continue for as long as the user
// has an in-progress training_draft — a DB lookup, not a callback match.
type draftContinuation struct {
	create trainingCreate
	drafts trainingDraft
}

func (h *draftContinuation) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	draft, err := h.drafts.GetDraftByUserID(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		return ErrSkip
	}
	if err != nil {
		return fmt.Errorf("get training draft: %w", err)
	}

	return h.create.Continue(ctx, draft, in)
}
