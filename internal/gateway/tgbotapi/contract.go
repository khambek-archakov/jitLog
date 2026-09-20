//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package tgbotapi

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// transport is what Gateway needs from a Telegram bot client — satisfied by
// *tgbotapi.BotAPI, but kept as an interface so Gateway doesn't depend on
// the concrete type (and can be tested with a fake).
type transport interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}
