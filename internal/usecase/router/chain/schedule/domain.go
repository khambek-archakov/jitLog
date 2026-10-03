// Package schedule groups every schedule:*-prefixed router trigger behind
// one Handler — the sibling of router/chain/training, same reasoning (see
// its package doc): only matches its own callback prefixes/tokens
// ("schedule:add", "schedule:view:", "schedule:edit:", "schedule:delete:",
// "schedule:list"), never imports chain, uses model.ErrSkip and its own
// private handler interface instead of chain.ErrSkip/chain.Handler.
package schedule

import (
	"context"
	"errors"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// Domain is every schedule:*-prefixed concern bundled behind one Handler.
type Domain struct {
	links sequence
}

func New(
	create scheduleCreate, list scheduleList, info scheduleInfo, update scheduleUpdate, del scheduleDelete,
) *Domain {
	return &Domain{
		links: sequence{
			&addTrigger{create: create},
			&viewTrigger{info: info},
			&editTrigger{update: update},
			&deleteTrigger{delete: del},
			&listTrigger{list: list},
		},
	}
}

func (d *Domain) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	return d.links.Handle(ctx, u, in)
}

// sequence runs its own members in order — duplicated from the parent
// chain package's own sequence type (not imported) to keep this package
// cycle-free.
type sequence []handler

func (s sequence) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	for _, h := range s {
		if err := h.Handle(ctx, u, in); !errors.Is(err, model.ErrSkip) {
			return err
		}
	}

	return model.ErrSkip
}
