//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package tgbotapi

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// transport is what Gateway needs from a Telegram bot client — satisfied by
// *tgbotapi.BotAPI, but kept as an interface so Gateway doesn't depend on
// the concrete type (and can be tested with a fake). Request alone covers
// every call Gateway makes — it never needs Send's extra step of parsing
// the response into a tgbotapi.Message, since no caller uses that value.
type transport interface {
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}
