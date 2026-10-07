// Package catalog groups every catalog:*-prefixed router trigger behind
// one Handler — the sibling of router/chain/training, router/chain/schedule
// and router/chain/competition, same reasoning (see training's package
// doc): only matches its own callback prefixes ("catalog:list",
// "catalog:list:page:", "catalog:city:", "catalog:view:", "catalog:add:"),
// never imports chain, uses model.ErrSkip and its own private handler
// interface instead of chain.ErrSkip/chain.Handler.
package catalog

import (
	"context"
	"errors"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// Domain is every catalog:*-prefixed concern bundled behind one Handler.
type Domain struct {
	links sequence
}

func New(list catalogList, info catalogInfo, add catalogAdd) *Domain {
	return &Domain{
		links: sequence{
			&listTrigger{list: list},
			&viewTrigger{info: info},
			&addTrigger{add: add},
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
