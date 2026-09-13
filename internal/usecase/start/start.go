package start

import (
	"context"
	"errors"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/steps"
)

// UseCase drives the /start onboarding scenario. It only speaks in
// dto.Input — tgbotapi.Update is translated into it at the delivery
// boundary (see internal/handler/update), so this package has no dependency
// on the Telegram transport.
type UseCase struct {
	bot      steps.Sender
	user     steps.User
	handlers map[model.OnboardingStep]steps.Handler
}

func New(bot steps.Sender, user steps.User) *UseCase {
	return &UseCase{
		bot:  bot,
		user: user,
		handlers: map[model.OnboardingStep]steps.Handler{
			model.OnboardingStepAwaitingName: steps.NewName(bot, user),
			model.OnboardingStepAwaitingAge:  steps.NewAge(bot, user),
			model.OnboardingStepAwaitingBelt: steps.NewBelt(bot, user),
			model.OnboardingStepCompleted:    steps.NewCompleted(bot),
		},
	}
}

func (uc *UseCase) Start(ctx context.Context, in dto.Input) error {
	u, err := uc.getOrCreateUser(ctx, in)
	if err != nil {
		return err
	}
	if u == nil {
		return nil
	}

	handler, ok := uc.handlers[u.OnboardingStep]
	if !ok {
		return nil
	}

	return handler.Handle(ctx, u, in)
}

func (uc *UseCase) getOrCreateUser(ctx context.Context, in dto.Input) (*model.User, error) {
	u, err := uc.user.GetByTelegramID(ctx, in.TelegramID)
	if err == nil {
		return u, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if !in.IsStartCmd {
		return nil, nil
	}

	u, err = uc.user.Create(ctx, in.TelegramID)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}
