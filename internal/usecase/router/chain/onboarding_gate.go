package chain

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// onboardingGate sends everything to onboarding until it's done — the
// user's current step decides, not the input's shape. There's no matching
// fallback type for the chain's own last link: onboarding's own
// Handle(ctx, *model.User, dto.Input) error signature already satisfies
// Handler exactly, so defaultOrder() uses the raw dependency directly there
// instead of wrapping it in an identity type.
type onboardingGate struct {
	onboarding onboarding
}

func (h *onboardingGate) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if u.OnboardingStep == model.OnboardingStepCompleted {
		return model.ErrSkip
	}

	return h.onboarding.Handle(ctx, u, in)
}
