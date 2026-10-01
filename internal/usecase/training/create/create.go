package create

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/create/steps"
)

type UseCase struct {
	repo     draftRepo
	handlers map[model.TrainingDraftStep]handler
}

func New(bot sender, repo draftRepo) *UseCase {
	return &UseCase{
		repo: repo,
		handlers: map[model.TrainingDraftStep]handler{
			model.TrainingDraftStepAwaitingDate:     steps.NewDate(bot, repo),
			model.TrainingDraftStepAwaitingType:     steps.NewType(bot, repo),
			model.TrainingDraftStepAwaitingDuration: steps.NewDuration(bot, repo),
			model.TrainingDraftStepAwaitingNotes:    steps.NewNotes(bot, repo),
		},
	}
}

func (uc *UseCase) Begin(ctx context.Context, userID int64, in dto.Input) error {
	d, err := uc.repo.CreateDraft(ctx, userID)
	if err != nil {
		return fmt.Errorf("create training draft: %w", err)
	}

	return uc.dispatch(ctx, d, in)
}

func (uc *UseCase) Continue(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	return uc.dispatch(ctx, d, in)
}

func (uc *UseCase) dispatch(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	h, ok := uc.handlers[d.Step]
	if !ok {
		return nil
	}

	return h.Handle(ctx, d, in)
}
