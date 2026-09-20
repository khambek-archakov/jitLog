package tgbotapi_test

import (
	"errors"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	gateway "github.com/khambek-archakov/jitLog/internal/gateway/tgbotapi"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
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
					Send(gomock.Any()).
					Return(tgbotapi.Message{}, nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "transport error",
			prepare: func(transport *Mocktransport) {
				transport.EXPECT().
					Send(gomock.Any()).
					Return(tgbotapi.Message{}, errors.New("fail"))
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

			err := g.Send(chatID, "hello")

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
					Send(gomock.Any()).
					DoAndReturn(func(c tgbotapi.Chattable) (tgbotapi.Message, error) {
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

						return tgbotapi.Message{}, nil
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
					Send(gomock.Any()).
					Return(tgbotapi.Message{}, errors.New("fail"))
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

			err := g.SendWithKeyboard(chatID, "pick one", keyboard)

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

			err := g.AnswerCallback(tc.callbackID)

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

			err := g.AnswerCallbackWithText(tc.callbackID, "Скоро!")

			tc.expected(t, err)
		})
	}
}
