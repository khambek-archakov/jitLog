// Package router is the only place in the usecase layer allowed to know
// that more than one scenario (onboarding, training) exists. Each scenario
// package stays oblivious to the others — Router just resolves the current
// user once and decides who handles the update.
package router

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
)

const callbackMenuAddTraining = "menu:add_training"

type Router struct {
	onboarding onboarding
	training   training
	user       user
	drafts     trainingDraft
}

func New(onboarding onboarding, training training, user user, drafts trainingDraft) *Router {
	return &Router{onboarding: onboarding, training: training, user: user, drafts: drafts}
}

func (r *Router) Route(ctx context.Context, in dto.Input) error {
	u, err := r.getOrCreateUser(ctx, in)
	if err != nil {
		return err
	}
	if u == nil {
		return nil
	}

	if u.OnboardingStep != model.OnboardingStepCompleted {
		return r.onboarding.Handle(ctx, u, in)
	}

	draft, err := r.drafts.GetDraftByUserID(ctx, u.ID)
	if err == nil {
		return r.training.Continue(ctx, draft, in)
	}
	if !errors.Is(err, model.ErrNotFound) {
		return fmt.Errorf("get training draft: %w", err)
	}

	if in.HasCallback && in.CallbackData == callbackMenuAddTraining {
		return r.training.Begin(ctx, u.ID, in)
	}

	return r.onboarding.Handle(ctx, u, in)
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
