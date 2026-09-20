package steps

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

const ageNotParsed = "Не смог разобрать точную цифру, поле оставлю пустым — всегда сможешь указать возраст позже."

const callbackSkipAge = "start:age:skip"

type AgeStep struct {
	bot  sender
	user user
}

func NewAge(bot sender, user user) *AgeStep {
	return &AgeStep{bot: bot, user: user}
}

func (s *AgeStep) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if in.IsStartCmd {
		return s.bot.SendWithKeyboard(in.ChatID, ageQuestion(*u.Name), ageKeyboard())
	}

	if in.HasCallback {
		switch in.CallbackData {
		case callbackBack:
			return s.handleBack(ctx, u, in)
		default:
			return s.handleSkip(ctx, u, in)
		}
	}

	if !in.HasMessage {
		return nil
	}

	return s.handleAnswer(ctx, u, in)
}

func (s *AgeStep) handleBack(ctx context.Context, u *model.User, in dto.Input) error {
	u.OnboardingStep = model.OnboardingStepAwaitingName

	if err := s.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return s.bot.Send(in.ChatID, askNameAgain)
}

func (s *AgeStep) handleSkip(ctx context.Context, u *model.User, in dto.Input) error {
	if in.CallbackData != callbackSkipAge {
		return s.bot.AnswerCallback(in.CallbackID)
	}

	u.OnboardingStep = model.OnboardingStepAwaitingBelt

	if err := s.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(in.ChatID, beltQuestion, beltKeyboard())
}

func (s *AgeStep) handleAnswer(ctx context.Context, u *model.User, in dto.Input) error {
	u.OnboardingStep = model.OnboardingStepAwaitingBelt

	age, parseErr := strconv.Atoi(strings.TrimSpace(in.Text))
	if parseErr != nil {
		if err := s.user.Update(ctx, u); err != nil {
			return fmt.Errorf("update user: %w", err)
		}

		if err := s.bot.Send(in.ChatID, ageNotParsed); err != nil {
			return err
		}

		return s.bot.SendWithKeyboard(in.ChatID, beltQuestion, beltKeyboard())
	}

	parsedAge := int16(age)
	u.Age = &parsedAge

	if err := s.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user age: %w", err)
	}

	return s.bot.SendWithKeyboard(in.ChatID, beltQuestion, beltKeyboard())
}

func ageQuestion(name string) string {
	return fmt.Sprintf(
		"Приятно познакомиться, %s! Перед тем, как начать, позволь задать тебе пару вопросов.\nСколько тебе лет?",
		name,
	)
}

func ageKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "Пропустить", Data: callbackSkipAge}),
		dto.Row(backButton()),
	}
}
