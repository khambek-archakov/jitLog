package tgbotapi_test

import (
	"context"
	"errors"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	gateway "github.com/khambek-archakov/jitLog/internal/gateway/tgbotapi"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const chatID int64 = 777

func TestGateway_Send(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prepare  func(transport *Mocktransport)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "success",
			prepare: func(transport *Mocktransport) {
				transport.EXPECT().
					Request(gomock.Any()).
					Return(&tgbotapi.APIResponse{}, nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "transport error",
			prepare: func(transport *Mocktransport) {
				transport.EXPECT().
					Request(gomock.Any()).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockTransport := NewMocktransport(ctrl)

			tc.prepare(mockTransport)

			g := gateway.New(mockTransport)

			err := g.Send(context.Background(), chatID, "hello")

			tc.expected(t, err)
		})
	}
}

func TestGateway_SendWithKeyboard(t *testing.T) {
	t.Parallel()

	keyboard := dto.Keyboard{
		dto.Row(dto.Button{Label: "Пропустить", Data: "skip"}),
		dto.Row(
			dto.Button{Label: "Белый", Data: "belt:white"},
			dto.Button{Label: "Синий", Data: "belt:blue"},
		),
	}

	tests := []struct {
		name     string
		prepare  func(transport *Mocktransport)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "keyboard is translated row by row",
			prepare: func(transport *Mocktransport) {
				transport.EXPECT().
					Request(gomock.Any()).
					DoAndReturn(func(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
						msg, ok := c.(tgbotapi.MessageConfig)
						require.True(t, ok)

						markup, ok := msg.ReplyMarkup.(tgbotapi.InlineKeyboardMarkup)
						require.True(t, ok)
						require.Len(t, markup.InlineKeyboard, 2)

						assert.Equal(t, "Пропустить", markup.InlineKeyboard[0][0].Text)
						assert.Equal(t, "skip", *markup.InlineKeyboard[0][0].CallbackData)

						assert.Equal(t, "Белый", markup.InlineKeyboard[1][0].Text)
						assert.Equal(t, "belt:white", *markup.InlineKeyboard[1][0].CallbackData)
						assert.Equal(t, "Синий", markup.InlineKeyboard[1][1].Text)
						assert.Equal(t, "belt:blue", *markup.InlineKeyboard[1][1].CallbackData)

						return &tgbotapi.APIResponse{}, nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "transport error",
			prepare: func(transport *Mocktransport) {
				transport.EXPECT().
					Request(gomock.Any()).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockTransport := NewMocktransport(ctrl)

			tc.prepare(mockTransport)

			g := gateway.New(mockTransport)

			err := g.SendWithKeyboard(context.Background(), chatID, "pick one", keyboard)

			tc.expected(t, err)
		})
	}
}

func TestGateway_EditMessageWithKeyboard(t *testing.T) {
	t.Parallel()

	const messageID = 555

	keyboard := dto.Keyboard{
		dto.Row(dto.Button{Label: "Отмена", Data: "cancel"}),
	}

	tests := []struct {
		name     string
		prepare  func(transport *Mocktransport)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "success",
			prepare: func(transport *Mocktransport) {
				transport.EXPECT().
					Request(gomock.Any()).
					DoAndReturn(func(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
						edit, ok := c.(tgbotapi.EditMessageTextConfig)
						require.True(t, ok)
						assert.Equal(t, chatID, edit.ChatID)
						assert.Equal(t, messageID, edit.MessageID)

						return &tgbotapi.APIResponse{}, nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "transport error",
			prepare: func(transport *Mocktransport) {
				transport.EXPECT().
					Request(gomock.Any()).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockTransport := NewMocktransport(ctrl)

			tc.prepare(mockTransport)

			g := gateway.New(mockTransport)

			err := g.EditMessageWithKeyboard(context.Background(), chatID, messageID, "new text", keyboard)

			tc.expected(t, err)
		})
	}
}

func TestGateway_AnswerCallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		callbackID string
		prepare    func(transport *Mocktransport)
		expected   func(t assert.TestingT, err error)
	}{
		{
			name:       "blank callback id is a no-op",
			callbackID: "",
			prepare:    func(transport *Mocktransport) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:       "success",
			callbackID: "cb-1",
			prepare: func(transport *Mocktransport) {
				transport.EXPECT().
					Request(gomock.Any()).
					Return(&tgbotapi.APIResponse{}, nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:       "transport error",
			callbackID: "cb-1",
			prepare: func(transport *Mocktransport) {
				transport.EXPECT().
					Request(gomock.Any()).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockTransport := NewMocktransport(ctrl)

			tc.prepare(mockTransport)

			g := gateway.New(mockTransport)

			err := g.AnswerCallback(context.Background(), tc.callbackID)

			tc.expected(t, err)
		})
	}
}

func TestGateway_AnswerCallbackWithText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		callbackID string
		prepare    func(transport *Mocktransport)
		expected   func(t assert.TestingT, err error)
	}{
		{
			name:       "blank callback id is a no-op",
			callbackID: "",
			prepare:    func(transport *Mocktransport) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:       "success",
			callbackID: "cb-1",
			prepare: func(transport *Mocktransport) {
				transport.EXPECT().
					Request(gomock.Any()).
					Return(&tgbotapi.APIResponse{}, nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:       "transport error",
			callbackID: "cb-1",
			prepare: func(transport *Mocktransport) {
				transport.EXPECT().
					Request(gomock.Any()).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockTransport := NewMocktransport(ctrl)

			tc.prepare(mockTransport)

			g := gateway.New(mockTransport)

			err := g.AnswerCallbackWithText(context.Background(), tc.callbackID, "Скоро!")

			tc.expected(t, err)
		})
	}
}

func TestGateway_RespectsContext(t *testing.T) {
	t.Parallel()

	t.Run("a cancelled context returns immediately instead of waiting for a stalled transport call", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)

		mockTransport := NewMocktransport(ctrl)

		started := make(chan struct{})
		release := make(chan struct{})
		t.Cleanup(func() { close(release) })

		mockTransport.EXPECT().
			Request(gomock.Any()).
			DoAndReturn(func(tgbotapi.Chattable) (*tgbotapi.APIResponse, error) {
				close(started) // prove the call genuinely started before ctx fires

				<-release // never sent during this test — simulates a stalled call

				return &tgbotapi.APIResponse{}, nil
			})

		g := gateway.New(mockTransport)

		ctx, cancel := context.WithCancel(context.Background())

		// Cancel only once the transport call is confirmed in-flight —
		// cancelling upfront would race the goroutine that calls
		// bot.Request, since ctx.Done() could already be selected before
		// that goroutine is even scheduled.
		go func() {
			<-started
			cancel()
		}()

		start := time.Now()
		err := g.Send(ctx, chatID, "hello")

		assert.ErrorIs(t, err, context.Canceled)
		assert.Less(t, time.Since(start), time.Second)
	})
}
