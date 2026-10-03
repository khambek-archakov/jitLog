// Package training groups every training:*-prefixed router trigger behind
// one Handler. It only ever matches its own callback prefixes
// ("menu:add_training", "training:view:", "training:history:page:",
// "training:edit:", "training:delete:") — nothing here ever fires for a
// "schedule:"/"stats:"/"profile:" callback — so the parent chain package
// can place this domain anywhere relative to schedule's own sibling
// package with no behavior change. This package deliberately never
// imports chain (that would cycle, since chain imports this package to
// build its own handler list); model.ErrSkip and its own private handler
// interface take the place of chain.ErrSkip/chain.Handler.
package training

import (
	"context"
	"errors"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// Domain is every training:*-prefixed concern bundled behind one Handler.
type Domain struct {
	links sequence
}

func New(
	create trainingCreate, info trainingInfo, history trainingHistory, update trainingUpdate, del trainingDelete,
) *Domain {
	return &Domain{
		links: sequence{
			&addTrigger{create: create},
			&viewTrigger{info: info},
			&historyTrigger{history: history},
			&editTrigger{update: update},
			&deleteTrigger{delete: del},
		},
	}
}

func (d *Domain) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	return d.links.Handle(ctx, u, in)
}

// sequence runs its own members in order — the same chain-of-responsibility
// mechanics the parent chain package's own sequence type uses, duplicated
// here (not imported) to keep this package cycle-free.
type sequence []handler

func (s sequence) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	for _, h := range s {
		if err := h.Handle(ctx, u, in); !errors.Is(err, model.ErrSkip) {
			return err
		}
	}

	return model.ErrSkip
}
