package tgbotapi

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

func toInlineKeyboard(k dto.Keyboard) tgbotapi.InlineKeyboardMarkup {
	rows := make([][]tgbotapi.InlineKeyboardButton, 0, len(k))

	for _, row := range k {
		buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(row))
		for _, b := range row {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(b.Label, b.Data))
		}

		rows = append(rows, buttons)
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}
