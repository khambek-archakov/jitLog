// Package tgbotapi is the gateway to the Telegram Bot API: the only place
// outside internal/handler/update allowed to import
// github.com/go-telegram-bot-api/telegram-bot-api/v5. Usecase code talks to
// it through Gateway's plain-typed methods (chat/callback IDs, strings,
// dto.Keyboard) and never sees a tgbotapi type.
package tgbotapi

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

// Gateway wraps a transport for sending messages and answering callbacks.
type Gateway struct {
	bot transport
}

func New(bot transport) *Gateway {
	return &Gateway{bot: bot}
}

// Send sends a plain text message.
func (g *Gateway) Send(chatID int64, text string) error {
	_, err := g.bot.Send(tgbotapi.NewMessage(chatID, text))
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

// SendWithKeyboard sends a text message with an inline keyboard attached.
func (g *Gateway) SendWithKeyboard(chatID int64, text string, keyboard dto.Keyboard) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = toInlineKeyboard(keyboard)

	_, err := g.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return nil
}

// AnswerCallback closes a pending callback query (stops the tap spinner)
// with no visible feedback. A blank callbackID is a no-op, so callers don't
// need to special-case updates that weren't callbacks.
func (g *Gateway) AnswerCallback(callbackID string) error {
	return g.AnswerCallbackWithText(callbackID, "")
}

// AnswerCallbackWithText closes a pending callback query and shows the user
// a short toast.
func (g *Gateway) AnswerCallbackWithText(callbackID, text string) error {
	if callbackID == "" {
		return nil
	}

	_, err := g.bot.Request(tgbotapi.NewCallback(callbackID, text))
	if err != nil {
		return fmt.Errorf("answer callback: %w", err)
	}

	return nil
}
