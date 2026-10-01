// Package router is the only place in the usecase layer allowed to know
// that more than one scenario (onboarding, the training sub-scenarios)
// exists. Each scenario package stays oblivious to the others — Router just
// resolves the current user once and decides who handles the update.
package router

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
)

const (
	callbackMenuAddTraining      = "menu:add_training"
	callbackTrainingViewPrefix   = "training:view:"
	callbackTrainingEditPrefix   = "training:edit:"
	callbackTrainingDeletePrefix = "training:delete:"
	callbackHistoryPagePrefix    = "training:history:page:"
)

type Router struct {
	onboarding onboarding
	create     trainingCreate
	info       trainingInfo
	history    trainingHistory
	update     trainingUpdate
	delete     trainingDelete
	user       user
	drafts     trainingDraft
	edits      trainingEditDraft
}

func New(
	onboarding onboarding,
	create trainingCreate,
	info trainingInfo,
	history trainingHistory,
	update trainingUpdate,
	del trainingDelete,
	user user,
	drafts trainingDraft,
	edits trainingEditDraft,
) *Router {
	return &Router{
		onboarding: onboarding, create: create, info: info, history: history, update: update, delete: del,
		user: user, drafts: drafts, edits: edits,
	}
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
		return r.create.Continue(ctx, draft, in)
	}
	if !errors.Is(err, model.ErrNotFound) {
		return fmt.Errorf("get training draft: %w", err)
	}

	editDraft, err := r.edits.GetEditDraftByUserID(ctx, u.ID)
	if err == nil {
		return r.update.Continue(ctx, editDraft, in)
	}
	if !errors.Is(err, model.ErrNotFound) {
		return fmt.Errorf("get training edit draft: %w", err)
	}

	if in.HasCallback {
		switch {
		case in.CallbackData == callbackMenuAddTraining:
			return r.create.Begin(ctx, u.ID, in)
		case strings.HasPrefix(in.CallbackData, callbackTrainingViewPrefix):
			return r.info.Handle(ctx, u.ID, in)
		case strings.HasPrefix(in.CallbackData, callbackHistoryPagePrefix):
			return r.history.Handle(ctx, u.ID, in)
		case strings.HasPrefix(in.CallbackData, callbackTrainingEditPrefix):
			return r.update.Handle(ctx, u.ID, in)
		case strings.HasPrefix(in.CallbackData, callbackTrainingDeletePrefix):
			return r.delete.Handle(ctx, u.ID, in)
		}
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
