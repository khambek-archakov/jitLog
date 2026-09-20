package steps

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

const (
	beltQuestion = "Какой у тебя пояс?"
	doneText     = "Спасибо! Итак, начнём 🥷\n\n" + menuPromptText
)

const (
	callbackBeltWhite  = "start:belt:white"
	callbackBeltBlue   = "start:belt:blue"
	callbackBeltPurple = "start:belt:purple"
	callbackBeltBrown  = "start:belt:brown"
	callbackBeltBlack  = "start:belt:black"
)

type BeltStep struct {
	bot  sender
	user user
}

func NewBelt(bot sender, user user) *BeltStep {
	return &BeltStep{bot: bot, user: user}
}

func (s *BeltStep) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if in.IsStartCmd {
		return s.bot.SendWithKeyboard(in.ChatID, beltQuestion, beltKeyboard())
	}

	if !in.HasCallback {
		return nil
	}

	if in.CallbackData == callbackBack {
		return s.handleBack(ctx, u, in)
	}

	belt, ok := beltFromCallback(in.CallbackData)
	if !ok {
		return s.bot.AnswerCallback(in.CallbackID)
	}

	u.Belt = belt
	u.OnboardingStep = model.OnboardingStepCompleted

	if err := s.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user belt: %w", err)
	}

	if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return sendMainMenu(s.bot, in.ChatID, doneText)
}

func (s *BeltStep) handleBack(ctx context.Context, u *model.User, in dto.Input) error {
	u.OnboardingStep = model.OnboardingStepAwaitingAge

	if err := s.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(in.ChatID, ageQuestion(*u.Name), ageKeyboard())
}

func beltKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "Белый", Data: callbackBeltWhite},
			dto.Button{Label: "Синий", Data: callbackBeltBlue},
		),
		dto.Row(
			dto.Button{Label: "Пурпурный", Data: callbackBeltPurple},
			dto.Button{Label: "Коричневый", Data: callbackBeltBrown},
		),
		dto.Row(
			dto.Button{Label: "Чёрный", Data: callbackBeltBlack},
		),
		dto.Row(backButton()),
	}
}

func beltFromCallback(data string) (model.Belt, bool) {
	switch data {
	case callbackBeltWhite:
		return model.BeltWhite, true
	case callbackBeltBlue:
		return model.BeltBlue, true
	case callbackBeltPurple:
		return model.BeltPurple, true
	case callbackBeltBrown:
		return model.BeltBrown, true
	case callbackBeltBlack:
		return model.BeltBlack, true
	default:
		return model.BeltNone, false
	}
}
