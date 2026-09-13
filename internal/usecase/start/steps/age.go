package steps

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

const ageNotParsed = "Не смог разобрать точную цифру, поле оставлю пустым — всегда сможешь указать возраст позже."

const callbackSkipAge = "start:age:skip"

type AgeStep struct {
	bot  Sender
	user User
}

func NewAge(bot Sender, user User) *AgeStep {
	return &AgeStep{bot: bot, user: user}
}

func (s *AgeStep) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if in.IsStartCmd {
		return sendWithKeyboard(s.bot, in.ChatID, ageQuestion(*u.Name), skipAgeKeyboard())
	}

	if in.HasCallback {
		return s.handleSkip(ctx, u, in)
	}

	if !in.HasMessage {
		return nil
	}

	return s.handleAnswer(ctx, u, in)
}

func (s *AgeStep) handleSkip(ctx context.Context, u *model.User, in dto.Input) error {
	if in.CallbackData != callbackSkipAge {
		return answerCallback(s.bot, in.CallbackID)
	}

	u.OnboardingStep = model.OnboardingStepAwaitingBelt

	if err := s.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if err := answerCallback(s.bot, in.CallbackID); err != nil {
		return err
	}

	return sendWithKeyboard(s.bot, in.ChatID, beltQuestion, beltKeyboard())
}

func (s *AgeStep) handleAnswer(ctx context.Context, u *model.User, in dto.Input) error {
	u.OnboardingStep = model.OnboardingStepAwaitingBelt

	age, parseErr := strconv.Atoi(strings.TrimSpace(in.Text))
	if parseErr != nil {
		if err := s.user.Update(ctx, u); err != nil {
			return fmt.Errorf("update user: %w", err)
		}

		if err := send(s.bot, in.ChatID, ageNotParsed); err != nil {
			return err
		}

		return sendWithKeyboard(s.bot, in.ChatID, beltQuestion, beltKeyboard())
	}

	parsedAge := int16(age)
	u.Age = &parsedAge

	if err := s.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user age: %w", err)
	}

	return sendWithKeyboard(s.bot, in.ChatID, beltQuestion, beltKeyboard())
}

func ageQuestion(name string) string {
	return fmt.Sprintf(
		"Приятно познакомиться, %s! Перед тем, как начать, позволь задать тебе пару вопросов.\nСколько тебе лет?",
		name,
	)
}

func skipAgeKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Пропустить", callbackSkipAge),
		),
	)
}
