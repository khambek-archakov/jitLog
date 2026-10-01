package history_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/history"
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
			name: "malformed page is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to list trainings",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:history:page:0"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().
					ListTrainings(gomock.Any(), userID, 5, 0).
					Return(nil, false, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "empty list on first page",
			in:   dto.Input{ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "training:history:page:0"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().
					ListTrainings(gomock.Any(), userID, 5, 0).
					Return(nil, false, nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), "Тренировок пока нет.", gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, keyboard dto.Keyboard) error {
						require.Len(t, keyboard, 1)
						assert.Equal(t, "menu:back", keyboard[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "first page with more after it shows a forward-only nav row",
			in:   dto.Input{ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "training:history:page:0"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().
					ListTrainings(gomock.Any(), userID, 5, 0).
					Return([]*model.Training{
						{ID: 7, Date: date, TrainingType: model.TrainingTypeNoGi, DurationMinutes: 90},
					}, true, nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), "📋 Мои тренировки", gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, keyboard dto.Keyboard) error {
						require.Len(t, keyboard, 3)

						assert.Equal(t, "training:view:7", keyboard[0][0].Data)
						assert.Contains(t, keyboard[0][0].Label, "30.09.2026")
						assert.Contains(t, keyboard[0][0].Label, "No-Gi")

						require.Len(t, keyboard[1], 1)
						assert.Equal(t, "›", keyboard[1][0].Label)
						assert.Equal(t, "training:history:page:1", keyboard[1][0].Data)

						assert.Equal(t, "menu:back", keyboard[2][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "middle page shows both nav directions",
			in:   dto.Input{ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "training:history:page:1"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().
					ListTrainings(gomock.Any(), userID, 5, 5).
					Return([]*model.Training{
						{ID: 3, Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 60},
					}, true, nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, keyboard dto.Keyboard) error {
						require.Len(t, keyboard[1], 2)
						assert.Equal(t, "‹", keyboard[1][0].Label)
						assert.Equal(t, "training:history:page:0", keyboard[1][0].Data)
						assert.Equal(t, "›", keyboard[1][1].Label)
						assert.Equal(t, "training:history:page:2", keyboard[1][1].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "last page shows only a back nav arrow",
			in:   dto.Input{ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "training:history:page:2"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().
					ListTrainings(gomock.Any(), userID, 5, 10).
					Return([]*model.Training{
						{ID: 1, Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 60},
					}, false, nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, keyboard dto.Keyboard) error {
						require.Len(t, keyboard[1], 1)
						assert.Equal(t, "‹", keyboard[1][0].Label)
						assert.Equal(t, "training:history:page:1", keyboard[1][0].Data)

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

			uc := history.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), userID, tc.in)

			tc.expected(t, err)
		})
	}
}
