package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// scheduleDraftContinuation hands off to scheduleCreate.Continue for as
// long as the user has an in-progress schedule_draft — a DB lookup, not a
// callback match, same reasoning as draftContinuation.
type scheduleDraftContinuation struct {
	create scheduleCreate
	drafts scheduleDraft
}

func (h *scheduleDraftContinuation) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	draft, err := h.drafts.GetDraftByUserID(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		return model.ErrSkip
	}
	if err != nil {
		return fmt.Errorf("get schedule draft: %w", err)
	}

	return h.create.Continue(ctx, draft, in)
}
