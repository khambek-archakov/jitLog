// Package router resolves the current user once per update and walks a
// chain of responsibility (see internal/usecase/router/chain) to decide
// which scenario handles it. Router itself knows nothing about onboarding
// or training — that knowledge, and the order it must be tried in, lives
// entirely in the chain package.
package router

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/router/chain"
)

type Router struct {
	handlers []chain.Handler
	user     user
}

func New(handlers []chain.Handler, user user) *Router {
	return &Router{handlers: handlers, user: user}
}

func (r *Router) Route(ctx context.Context, in dto.Input) error {
	u, err := r.getOrCreateUser(ctx, in)
	if err != nil {
		return err
	}
	if u == nil {
		return nil
	}

	for _, h := range r.handlers {
		if err := h.Handle(ctx, u, in); !errors.Is(err, chain.ErrSkip) {
			return err
		}
	}

	return nil
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
