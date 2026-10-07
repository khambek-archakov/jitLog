package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// scheduleEditDraftContinuation mirrors editDraftContinuation for
// schedule/update's own pending free-text edit (time's "Другое"). It only
// ever claims a message — a callback (the prompt's own "❌ Отмена" button)
// is left for schedule's own schedule:edit:* trigger to dispatch to Handle
// instead, since Continue only understands free text.
type scheduleEditDraftContinuation struct {
	update scheduleUpdate
	edits  scheduleEditDraft
}

func (h *scheduleEditDraftContinuation) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if in.HasCallback {
		return model.ErrSkip
	}

	draft, err := h.edits.GetEditDraftByUserID(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		return model.ErrSkip
	}
	if err != nil {
		return fmt.Errorf("get schedule edit draft: %w", err)
	}

	return h.update.Continue(ctx, draft, in)
}
