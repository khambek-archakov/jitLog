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

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

type Router struct {
	chain chain
	user  user
}

func New(c chain, user user) *Router {
	return &Router{chain: c, user: user}
}

func (r *Router) Route(ctx context.Context, in dto.Input) error {
	u, err := r.getOrCreateUser(ctx, in)
	if err != nil {
		return err
	}
	if u == nil {
		return nil
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
