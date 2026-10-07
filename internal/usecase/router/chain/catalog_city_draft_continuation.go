package chain

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// catalogCityDraftContinuation hands off to list.Continue for as long as
// the user has a pending catalog city-filter prompt open — a DB lookup,
// not a callback match, same shape as every other draft continuation in
// this file.
type catalogCityDraftContinuation struct {
	list   catalogList
	drafts catalogCityDraft
}

func (h *catalogCityDraftContinuation) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	draft, err := h.drafts.GetCityDraftByUserID(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		return model.ErrSkip
	}
	if err != nil {
		return fmt.Errorf("get catalog city draft: %w", err)
	}

	return h.list.Continue(ctx, u, draft, in)
}
