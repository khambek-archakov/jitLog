package chain

import (
	"context"
	"errors"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// sequence runs its own members in order — the same chain-of-responsibility
// mechanics Chain.Handle itself uses to walk defaultOrder(), just one level
// down, so a whole group of links can be handed to defaultOrder() as a single
// Handler. If every member skips, sequence itself reports model.ErrSkip so
// the outer list can move on to whatever comes after it.
type sequence []Handler

func (s sequence) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	for _, h := range s {
		if err := h.Handle(ctx, u, in); !errors.Is(err, model.ErrSkip) {
			return err
		}
	}

	return model.ErrSkip
}
