// Package create owns the "➕ Добавить" wizard for a recurring schedule
// slot: day of week → time → training type. Mirrors
// internal/usecase/training/create's shape closely (same step-dispatch-map
// pattern, same centralized cancel handling).
package create

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/menu"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/create/steps"
)

// callbackCancel mirrors steps' own private constant — every step shows
// the button, but it's handled here, once.
const callbackCancel = "schedule:draft:cancel"

type UseCase struct {
	bot      sender
	repo     draftRepo
	handlers map[model.ScheduleDraftStep]handler
}

func New(bot sender, repo draftRepo) *UseCase {
	return &UseCase{
		bot:  bot,
		repo: repo,
		handlers: map[model.ScheduleDraftStep]handler{
			model.ScheduleDraftStepAwaitingDay:  steps.NewDay(bot, repo),
			model.ScheduleDraftStepAwaitingTime: steps.NewTime(bot, repo),
			model.ScheduleDraftStepAwaitingType: steps.NewType(bot, repo),
		},
	}
}

func (uc *UseCase) Begin(ctx context.Context, userID int64, in dto.Input) error {
	d, err := uc.repo.CreateDraft(ctx, userID)
	if err != nil {
		return fmt.Errorf("create schedule draft: %w", err)
	}

	return uc.dispatch(ctx, d, in)
}

func (uc *UseCase) Continue(ctx context.Context, d *model.ScheduleDraft, in dto.Input) error {
	return uc.dispatch(ctx, d, in)
}

func (uc *UseCase) dispatch(ctx context.Context, d *model.ScheduleDraft, in dto.Input) error {
	if in.HasCallback && in.CallbackData == callbackCancel {
		return uc.cancel(ctx, d, in)
	}

	h, ok := uc.handlers[d.Step]
	if !ok {
		return nil
	}

	return h.Handle(ctx, d, in)
}

func (uc *UseCase) cancel(ctx context.Context, d *model.ScheduleDraft, in dto.Input) error {
	if err := uc.repo.DeleteDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete schedule draft: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, menu.Text, menu.Keyboard())
}
