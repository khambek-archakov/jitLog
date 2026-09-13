package update

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

// NewInput translates a raw Telegram update into the onboarding usecase's
// transport-neutral dto.Input. This is the only place allowed to know about
// tgbotapi.Update.
func NewInput(update tgbotapi.Update) dto.Input {
	in := dto.Input{ChatID: chatID(update)}

	switch {
	case update.Message != nil:
		in.TelegramID = update.Message.From.ID
		in.HasMessage = true
		in.Text = update.Message.Text
		in.IsStartCmd = update.Message.IsCommand() && update.Message.Command() == "start"
	case update.CallbackQuery != nil:
		in.TelegramID = update.CallbackQuery.From.ID
		in.HasCallback = true
		in.CallbackID = update.CallbackQuery.ID
		in.CallbackData = update.CallbackQuery.Data
	}

	return in
}

func chatID(update tgbotapi.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	}

	if update.CallbackQuery != nil {
		return update.CallbackQuery.Message.Chat.ID
	}

	return 0
}
