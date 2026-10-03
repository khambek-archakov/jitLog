package create

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/menu"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/create/steps"
)

// callbackCancel mirrors steps' own private constant — every step shows
// the button, but it's handled here, once, rather than duplicated into
// each step's own business logic.
const callbackCancel = "training:cancel"

type UseCase struct {
	bot      sender
	repo     draftRepo
	handlers map[model.TrainingDraftStep]handler
}

func New(bot sender, repo draftRepo) *UseCase {
	return &UseCase{
		bot:  bot,
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
	if in.HasCallback && in.CallbackData == callbackCancel {
		return uc.cancel(ctx, d, in)
	}

	h, ok := uc.handlers[d.Step]
	if !ok {
		return nil
	}

	return h.Handle(ctx, d, in)
}

func (uc *UseCase) cancel(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	if err := uc.repo.DeleteDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete training draft: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, menu.Text, menu.Keyboard())
}
