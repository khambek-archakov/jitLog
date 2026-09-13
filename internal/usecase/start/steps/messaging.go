package steps

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func answerCallback(bot Sender, callbackID string) error {
	if callbackID == "" {
		return nil
	}

	_, err := bot.Request(tgbotapi.NewCallback(callbackID, ""))
	if err != nil {
		return fmt.Errorf("answer callback: %w", err)
	}

	return nil
}

func send(bot Sender, chatID int64, text string) error {
	_, err := bot.Send(tgbotapi.NewMessage(chatID, text))
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func sendWithKeyboard(bot Sender, chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = keyboard

	_, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

func sendMainMenu(bot Sender, chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Добавить тренировку")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Расписание")),
		tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton("Статистика")),
	)

	_, err := bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}
