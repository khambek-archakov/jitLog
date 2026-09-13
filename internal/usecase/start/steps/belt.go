package steps

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

const (
	beltQuestion = "Какой у тебя пояс?"
	doneText     = "Спасибо! Итак, начнём 🥷"
)

const (
	callbackBeltWhite  = "start:belt:white"
	callbackBeltBlue   = "start:belt:blue"
	callbackBeltPurple = "start:belt:purple"
	callbackBeltBrown  = "start:belt:brown"
	callbackBeltBlack  = "start:belt:black"
)

type BeltStep struct {
	bot  Sender
	user User
}

func NewBelt(bot Sender, user User) *BeltStep {
	return &BeltStep{bot: bot, user: user}
}

func (s *BeltStep) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if in.IsStartCmd {
		return sendWithKeyboard(s.bot, in.ChatID, beltQuestion, beltKeyboard())
	}

	if !in.HasCallback {
		return nil
	}

	belt, ok := beltFromCallback(in.CallbackData)
	if !ok {
		return answerCallback(s.bot, in.CallbackID)
	}

	u.Belt = belt
	u.OnboardingStep = model.OnboardingStepCompleted

	if err := s.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user belt: %w", err)
	}

	if err := answerCallback(s.bot, in.CallbackID); err != nil {
		return err
	}

	return sendMainMenu(s.bot, in.ChatID, doneText)
}

func beltKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Белый", callbackBeltWhite),
			tgbotapi.NewInlineKeyboardButtonData("Синий", callbackBeltBlue),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Фиолетовый", callbackBeltPurple),
			tgbotapi.NewInlineKeyboardButtonData("Коричневый", callbackBeltBrown),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Чёрный", callbackBeltBlack),
		),
	)
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
