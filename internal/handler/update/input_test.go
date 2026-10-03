package update_test

import (
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"

	"github.com/khambek-archakov/jitLog/internal/handler/update"
)

func TestNewInput(t *testing.T) {
	t.Parallel()

	t.Run("message", func(t *testing.T) {
		t.Parallel()

		upd := tgbotapi.Update{
			Message: &tgbotapi.Message{
				MessageID: 5,
				From:      &tgbotapi.User{ID: 42},
				Chat:      &tgbotapi.Chat{ID: 777},
				Text:      "hi",
			},
		}

		in := update.NewInput(upd)

		assert.Equal(t, int64(42), in.TelegramID)
		assert.Equal(t, int64(777), in.ChatID)
		assert.Equal(t, 5, in.MessageID)
		assert.True(t, in.HasMessage)
		assert.Equal(t, "hi", in.Text)
		assert.False(t, in.HasCallback)
	})

	t.Run("/start command", func(t *testing.T) {
		t.Parallel()

		upd := tgbotapi.Update{
			Message: &tgbotapi.Message{
				MessageID: 1,
				From:      &tgbotapi.User{ID: 42},
				Chat:      &tgbotapi.Chat{ID: 777},
				Text:      "/start",
				Entities:  []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: 6}},
			},
		}

		in := update.NewInput(upd)

		assert.True(t, in.IsStartCmd)
	})

	t.Run("callback with its originating message present", func(t *testing.T) {
		t.Parallel()

		upd := tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				ID:      "cb-1",
				From:    &tgbotapi.User{ID: 42},
				Data:    "schedule:list",
				Message: &tgbotapi.Message{MessageID: 9, Chat: &tgbotapi.Chat{ID: 777}},
			},
		}

		in := update.NewInput(upd)

		assert.Equal(t, int64(42), in.TelegramID)
		assert.Equal(t, int64(777), in.ChatID)
		assert.Equal(t, 9, in.MessageID)
		assert.True(t, in.HasCallback)
		assert.Equal(t, "cb-1", in.CallbackID)
		assert.Equal(t, "schedule:list", in.CallbackData)
	})

	t.Run("callback whose originating message Telegram omitted — must not panic", func(t *testing.T) {
		t.Parallel()

		upd := tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				ID:      "cb-1",
				From:    &tgbotapi.User{ID: 42},
				Data:    "schedule:list",
				Message: nil,
			},
		}

		assert.NotPanics(t, func() {
			in := update.NewInput(upd)

			assert.Equal(t, int64(42), in.TelegramID)
			assert.Equal(t, int64(0), in.ChatID)
			assert.Equal(t, 0, in.MessageID)
			assert.True(t, in.HasCallback)
			assert.Equal(t, "cb-1", in.CallbackID)
			assert.Equal(t, "schedule:list", in.CallbackData)
		})
	})

	t.Run("neither message nor callback — zero value", func(t *testing.T) {
		t.Parallel()

		in := update.NewInput(tgbotapi.Update{})

		assert.Equal(t, int64(0), in.ChatID)
		assert.False(t, in.HasMessage)
		assert.False(t, in.HasCallback)
	})
}
