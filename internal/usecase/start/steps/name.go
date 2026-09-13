package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

const (
	greetingText = "Привет! Я JitLog — твой личный дневник тренировок по джиу-джитсу 🥋\nКак тебя зовут?"
	askNameAgain = "Как тебя зовут?"
)

type NameStep struct {
	bot  Sender
	user User
}

func NewName(bot Sender, user User) *NameStep {
	return &NameStep{bot: bot, user: user}
}

func (s *NameStep) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if in.IsStartCmd {
		return send(s.bot, in.ChatID, greetingText)
	}

	if !in.HasMessage {
		return answerCallback(s.bot, in.CallbackID)
	}

	name := strings.TrimSpace(in.Text)
	if name == "" {
		return send(s.bot, in.ChatID, askNameAgain)
	}

	u.Name = &name
	u.OnboardingStep = model.OnboardingStepAwaitingAge

	if err := s.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user name: %w", err)
	}

	return sendWithKeyboard(s.bot, in.ChatID, ageQuestion(name), skipAgeKeyboard())
}
