// Package tgbotapi is the gateway to the Telegram Bot API: the only place
// outside internal/handler/update allowed to import
// github.com/go-telegram-bot-api/telegram-bot-api/v5. Usecase code talks to
// it through Gateway's plain-typed methods (chat/callback IDs, strings,
// dto.Keyboard) and never sees a tgbotapi type.
package tgbotapi

import (
	"context"
	"errors"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// telegramForbidden is the Bot API's own error code for "bot was blocked
// by the user" (and a few other access-revoked cases Telegram folds into
// the same code) — classified here, at the one place allowed to see a raw
// tgbotapi.Error, into model.ErrUserBlockedBot so every other package only
// ever deals with a transport-agnostic sentinel.
const telegramForbidden = 403

// classify turns a raw transport error into model.ErrUserBlockedBot when
// it's Telegram reporting code 403, leaving every other error (network
// failures, other API error codes) untouched.
func classify(err error) error {
	var tgErr *tgbotapi.Error
	if errors.As(err, &tgErr) && tgErr.Code == telegramForbidden {
		return model.ErrUserBlockedBot
	}

	return err
}

// Gateway wraps a transport for sending messages and answering callbacks.
type Gateway struct {
	bot transport
}

func New(bot transport) *Gateway {
	return &Gateway{bot: bot}
}

// Send sends a plain text message.
func (g *Gateway) Send(ctx context.Context, chatID int64, text string) error {
	_, err := request(ctx, g.bot, tgbotapi.NewMessage(chatID, text))
	if err != nil {
		return fmt.Errorf("send message: %w", classify(err))
	}

	return nil
}

// SendWithKeyboard sends a text message with an inline keyboard attached.
func (g *Gateway) SendWithKeyboard(ctx context.Context, chatID int64, text string, keyboard dto.Keyboard) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = toInlineKeyboard(keyboard)

	_, err := request(ctx, g.bot, msg)
	if err != nil {
		return fmt.Errorf("send message: %w", classify(err))
	}

	return nil
}

// EditMessageWithKeyboard rewrites an already-sent message's text and
// keyboard in place (e.g. paging a calendar) instead of sending a new one.
func (g *Gateway) EditMessageWithKeyboard(
	ctx context.Context, chatID int64, messageID int, text string, keyboard dto.Keyboard,
) error {
	edit := tgbotapi.NewEditMessageTextAndMarkup(chatID, messageID, text, toInlineKeyboard(keyboard))

	_, err := request(ctx, g.bot, edit)
	if err != nil {
		return fmt.Errorf("edit message: %w", classify(err))
	}

	return nil
}

// SendAnimationWithKeyboard sends an animation by its Telegram file ID,
// with a caption and an inline keyboard attached.
func (g *Gateway) SendAnimationWithKeyboard(
	ctx context.Context, chatID int64, fileID, caption string, keyboard dto.Keyboard,
) error {
	anim := tgbotapi.NewAnimation(chatID, tgbotapi.FileID(fileID))
	anim.Caption = caption
	anim.ReplyMarkup = toInlineKeyboard(keyboard)

	_, err := request(ctx, g.bot, anim)
	if err != nil {
		return fmt.Errorf("send animation: %w", classify(err))
	}

	return nil
}

// RemoveKeyboard strips an already-sent message's inline keyboard in
// place, leaving its text/caption untouched — editMessageReplyMarkup works
// the same whether messageID is a plain text message or a media message
// with a caption, so this covers both.
func (g *Gateway) RemoveKeyboard(ctx context.Context, chatID int64, messageID int) error {
	edit := tgbotapi.NewEditMessageReplyMarkup(chatID, messageID, tgbotapi.NewInlineKeyboardMarkup())

	_, err := request(ctx, g.bot, edit)
	if err != nil {
		return fmt.Errorf("remove keyboard: %w", classify(err))
	}

	return nil
}

// AnswerCallback closes a pending callback query (stops the tap spinner)
// with no visible feedback. A blank callbackID is a no-op, so callers don't
// need to special-case updates that weren't callbacks.
func (g *Gateway) AnswerCallback(ctx context.Context, callbackID string) error {
	return g.AnswerCallbackWithText(ctx, callbackID, "")
}

// AnswerCallbackWithText closes a pending callback query and shows the user
// a short toast.
func (g *Gateway) AnswerCallbackWithText(ctx context.Context, callbackID, text string) error {
	if callbackID == "" {
		return nil
	}

	_, err := request(ctx, g.bot, tgbotapi.NewCallback(callbackID, text))
	if err != nil {
		return fmt.Errorf("answer callback: %w", err)
	}

	return nil
}

// request runs c through bot in its own goroutine and races it against
// ctx. tgbotapi's own HTTP client has no context support at all (see
// transport's doc comment) — it can't be told to abort an in-flight
// request — so this can't cancel the call itself. What it does guarantee:
// the caller (and whatever semaphore slot or deadline it's holding, see
// internal/handler/update) is released the moment ctx says to stop,
// instead of blocking for however long the stalled HTTP round-trip takes.
// The abandoned goroutine still exits on its own once that round-trip
// completes or the client's own timeout fires (main.go sets one); its
// result is simply discarded.
func request(ctx context.Context, bot transport, c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
	type result struct {
		resp *tgbotapi.APIResponse
		err  error
	}

	done := make(chan result, 1)

	go func() {
		resp, err := bot.Request(c)
		done <- result{resp: resp, err: err}
	}()

	select {
	case r := <-done:
		return r.resp, r.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
