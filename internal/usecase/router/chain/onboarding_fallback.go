package chain

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// onboardingFallback is the chain's last link — anything no earlier link
// claimed falls back to onboarding's own menu handling, so it never
// returns ErrSkip.
type onboardingFallback struct {
	onboarding onboarding
}

func (h *onboardingFallback) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	return h.onboarding.Handle(ctx, u, in)
}
