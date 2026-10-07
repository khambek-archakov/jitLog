package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	greetingText = "Привет! Я JitLog — бот для бразильского джиу-джитсу 🥋\nКак тебя зовут?"
	askNameAgain = "Как тебя зовут?"
)

type NameStep struct {
	bot  sender
	user user
}

func NewName(bot sender, user user) *NameStep {
	return &NameStep{bot: bot, user: user}
}

func (s *NameStep) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if in.IsStartCmd {
		return s.bot.Send(ctx, in.ChatID, greetingText)
	}

	if !in.HasMessage {
		return s.bot.AnswerCallback(ctx, in.CallbackID)
	}

	name := strings.TrimSpace(in.Text)
	if name == "" {
		return s.bot.Send(ctx, in.ChatID, askNameAgain)
	}

	u.Name = &name
	u.OnboardingStep = model.OnboardingStepAwaitingBelt

	if err := s.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user name: %w", err)
	}

	return s.bot.SendWithKeyboard(ctx, in.ChatID, beltQuestion, beltKeyboard())
}
