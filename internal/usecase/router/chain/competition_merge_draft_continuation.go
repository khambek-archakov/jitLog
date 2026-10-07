package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// competitionMergeDraftContinuation hands off to moderate.Continue for as
// long as the admin has a pending "🔗 Это дубль" title search in
// progress — a DB lookup, not a callback match, same shape as every other
// draft continuation in this file.
type competitionMergeDraftContinuation struct {
	moderate competitionModerate
	drafts   competitionMergeDraft
}

func (h *competitionMergeDraftContinuation) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	draft, err := h.drafts.GetMergeDraftByUserID(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		return model.ErrSkip
	}
	if err != nil {
		return fmt.Errorf("get competition merge draft: %w", err)
	}

	return h.moderate.Continue(ctx, u, draft, in)
}
