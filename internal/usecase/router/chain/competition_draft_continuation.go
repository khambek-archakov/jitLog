package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// competitionDraftContinuation hands off to create.Continue for as long as
// the user has an in-progress user_competition_draft — a DB lookup, not a
// callback match.
type competitionDraftContinuation struct {
	create competitionCreate
	drafts competitionDraft
}

func (h *competitionDraftContinuation) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	draft, err := h.drafts.GetDraftByUserID(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		return model.ErrSkip
	}
	if err != nil {
		return fmt.Errorf("get user competition draft: %w", err)
	}

	return h.create.Continue(ctx, u, draft, in)
}
