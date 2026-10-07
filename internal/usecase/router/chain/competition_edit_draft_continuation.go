package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// competitionEditDraftContinuation mirrors profileEditDraftContinuation for
// update's own pending free-text edit — unlike training/schedule, every
// field here (title, date, end date, city, url, result) goes through this
// path, not just a couple of leaves. It only ever claims a message — a
// callback (the prompt's own "❌ Отмена" button) is left for competition's
// own competition:edit:* trigger to dispatch to Handle instead, since
// Continue only understands free text.
type competitionEditDraftContinuation struct {
	update competitionUpdate
	edits  competitionEditDraft
}

func (h *competitionEditDraftContinuation) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if in.HasCallback {
		return model.ErrSkip
	}

	draft, err := h.edits.GetEditDraftByUserID(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		return model.ErrSkip
	}
	if err != nil {
		return fmt.Errorf("get user competition edit draft: %w", err)
	}

	return h.update.Continue(ctx, u, draft, in)
}
