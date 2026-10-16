// Package router resolves the current user once per update and hands it to
// the assembled chain of responsibility (see internal/usecase/router/chain)
// to decide which scenario handles it. Router itself knows nothing about
// onboarding or training, nor even that "the chain" is a walked list at
// all — that's entirely chain.Chain's own concern, reached through its one
// exported Handle method.
package router

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

type Router struct {
	chain chain
	user  user
	now   func() time.Time
}

func New(c chain, user user) *Router {
	return NewWithClock(c, user, time.Now)
}

// NewWithClock lets tests substitute a fixed instant instead of the real
// wall clock for the activity middleware (see activity.go) — same pattern
// as internal/usecase/reminder.NewWithClock.
func NewWithClock(c chain, user user, now func() time.Time) *Router {
	return &Router{chain: c, user: user, now: now}
}

// Route resolves the current user, records their activity (see
// touchActivity in activity.go — this is the one place every incoming
// update passes through, so it's the one place that needs to), and hands
// off to the chain.
func (r *Router) Route(ctx context.Context, in dto.Input) error {
	u, err := r.getOrCreateUser(ctx, in)
	if err != nil {
		return err
	}
	if u == nil {
		return nil
	}

	if err := r.touchActivity(ctx, u); err != nil {
		return err
	}

	return r.chain.Handle(ctx, u, in)
}

func (r *Router) getOrCreateUser(ctx context.Context, in dto.Input) (*model.User, error) {
	u, err := r.user.GetByTelegramID(ctx, in.TelegramID)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, fmt.Errorf("get user: %w", err)
	}

	if !in.IsStartCmd {
		return nil, nil
	}

	u, err = r.user.Create(ctx, in.TelegramID)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}
