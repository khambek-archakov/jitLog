package onboarding

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/steps"
)

type UseCase struct {
	handlers map[model.OnboardingStep]handler
}

func New(bot sender, user user) *UseCase {
	return &UseCase{
		handlers: map[model.OnboardingStep]handler{
			model.OnboardingStepAwaitingName: steps.NewName(bot, user),
			model.OnboardingStepAwaitingAge:  steps.NewAge(bot, user),
			model.OnboardingStepAwaitingBelt: steps.NewBelt(bot, user),
			model.OnboardingStepCompleted:    steps.NewCompleted(bot),
		},
	}
}

func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	h, ok := uc.handlers[u.OnboardingStep]
	if !ok {
		return nil
	}

	return h.Handle(ctx, u, in)
}
