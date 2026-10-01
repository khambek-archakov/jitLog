package info_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/info"
)

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	date := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MocktrainingRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no callback — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed id is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:view:nope"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "training not found",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:view:7"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().
					GetTraining(gomock.Any(), int64(7)).
					Return(nil, model.ErrNotFound)

				sender.EXPECT().
					AnswerCallbackWithText("cb-1", "Тренировка не найдена.").
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "training belongs to someone else",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:view:7"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().
					GetTraining(gomock.Any(), int64(7)).
					Return(&model.Training{ID: 7, UserID: userID + 1}, nil)

				sender.EXPECT().
					AnswerCallbackWithText("cb-1", "Тренировка не найдена.").
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch training",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:view:7"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().
					GetTraining(gomock.Any(), int64(7)).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "shows the card in place",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "training:view:7",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().
					GetTraining(gomock.Any(), int64(7)).
					Return(&model.Training{
						ID: 7, UserID: userID, Date: date,
						TrainingType: model.TrainingTypeNoGi, DurationMinutes: 90,
					}, nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ int64, _ int, text string, keyboard dto.Keyboard) error {
						assert.Contains(t, text, "Тренировка")
						assert.Contains(t, text, "30 сентября 2026")
						assert.Contains(t, text, "🥷 No-Gi")
						assert.Contains(t, text, "90 минут")

						assert.Equal(t, "training:edit:7", keyboard[0][0].Data)
						assert.Equal(t, "training:delete:7", keyboard[1][0].Data)
						assert.Equal(t, "training:history:page:0", keyboard[2][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockSender := NewMocksender(ctrl)
			mockRepo := NewMocktrainingRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := info.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), userID, tc.in)

			tc.expected(t, err)
		})
	}
}
