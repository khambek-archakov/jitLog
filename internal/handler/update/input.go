package update

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// NewInput translates a raw Telegram update into the onboarding usecase's
// transport-neutral dto.Input. This is the only place allowed to know about
// tgbotapi.Update.
func NewInput(update tgbotapi.Update) dto.Input {
	in := dto.Input{ChatID: chatID(update)}

	switch {
	case update.Message != nil:
		in.TelegramID = update.Message.From.ID
		in.MessageID = update.Message.MessageID
		in.HasMessage = true
		in.Text = update.Message.Text
		in.IsStartCmd = update.Message.IsCommand() && update.Message.Command() == "start"
	case update.CallbackQuery != nil:
		in.TelegramID = update.CallbackQuery.From.ID
		in.HasCallback = true
		in.CallbackID = update.CallbackQuery.ID
		in.CallbackData = update.CallbackQuery.Data

		// Telegram omits CallbackQuery.Message when the original message
		// is too old, already deleted, or the callback came from an
		// inline-mode result — MessageID just isn't available then.
		if update.CallbackQuery.Message != nil {
			in.MessageID = update.CallbackQuery.Message.MessageID
		}
	}

	return in
}

func chatID(update tgbotapi.Update) int64 {
	if update.Message != nil {
		return update.Message.Chat.ID
	}

	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		return update.CallbackQuery.Message.Chat.ID
	}

	return 0
}

// telegramID extracts the sender's id straight from the raw update — used
// to pick this update's per-user serialization lock (see Handler.lockFor)
// before NewInput's own fuller translation runs inside the recover-guarded
// goroutine.
func telegramID(update tgbotapi.Update) int64 {
	if update.Message != nil {
		return update.Message.From.ID
	}

	if update.CallbackQuery != nil {
		return update.CallbackQuery.From.ID
	}

	return 0
}
