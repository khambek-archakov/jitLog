package chain

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// onboardingGate sends everything to onboarding until it's done — the
// user's current step decides, not the input's shape.
type onboardingGate struct {
	onboarding onboarding
}

func (h *onboardingGate) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if u.OnboardingStep == model.OnboardingStepCompleted {
		return ErrSkip
	}

	return h.onboarding.Handle(ctx, u, in)
}
