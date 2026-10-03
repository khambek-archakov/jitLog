package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// scheduleEditDraftContinuation mirrors editDraftContinuation for
// schedule/update's own pending free-text edit (time's "Другое").
type scheduleEditDraftContinuation struct {
	update scheduleUpdate
	edits  scheduleEditDraft
}

func (h *scheduleEditDraftContinuation) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	draft, err := h.edits.GetEditDraftByUserID(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		return model.ErrSkip
	}
	if err != nil {
		return fmt.Errorf("get schedule edit draft: %w", err)
	}

	return h.update.Continue(ctx, draft, in)
}
