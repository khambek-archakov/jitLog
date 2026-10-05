package steps

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// AgeStep no longer asks anything — the age question was removed from
// onboarding (age is now editable from the profile screen instead). It's
// kept only so a user whose onboarding_step is still stuck at
// awaiting_age (e.g. mid-flow when this shipped) gets silently forwarded
// to the belt step on their next message, instead of being stranded.
type AgeStep struct {
	bot  sender
	user user
}

func NewAge(bot sender, user user) *AgeStep {
	return &AgeStep{bot: bot, user: user}
}

func (s *AgeStep) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	u.OnboardingStep = model.OnboardingStepAwaitingBelt

	if err := s.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return s.bot.SendWithKeyboard(ctx, in.ChatID, beltQuestion, beltKeyboard())
}
