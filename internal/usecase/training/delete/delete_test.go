package delete_test

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
	delete "github.com/khambek-archakov/jitLog/internal/usecase/training/delete"
)

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	training := &model.Training{ID: 7, UserID: userID, Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 60}

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
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:delete:nope"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "training not found shows confirm screen path fails gracefully",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:delete:7"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				sender.EXPECT().AnswerCallbackWithText("cb-1", "Тренировка не найдена.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "training belongs to someone else",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:delete:7"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				other := *training
				other.UserID = userID + 1
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(&other, nil)
				sender.EXPECT().AnswerCallbackWithText("cb-1", "Тренировка не найдена.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch training",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:delete:7"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "shows a confirm screen",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "training:delete:7",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(training, nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Удалить тренировку?")
						assert.Equal(t, "training:delete:confirm:7", kb[0][0].Data)
						assert.Equal(t, "training:view:7", kb[1][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "confirm deletes and shows a back-to-list button",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "training:delete:confirm:7",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(training, nil)
				repo.EXPECT().DeleteTraining(gomock.Any(), int64(7)).Return(nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), "🗑 Тренировка удалена.", gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 1)
						assert.Equal(t, "training:history:page:0", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "confirm on someone else's training is refused",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:delete:confirm:7",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				other := *training
				other.UserID = userID + 1
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(&other, nil)
				sender.EXPECT().AnswerCallbackWithText("cb-1", "Тренировка не найдена.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to delete training",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:delete:confirm:7",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().GetTraining(gomock.Any(), int64(7)).Return(training, nil)
				repo.EXPECT().DeleteTraining(gomock.Any(), int64(7)).Return(errors.New("fail"))
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

			mockSender := NewMocksender(ctrl)
			mockRepo := NewMocktrainingRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := delete.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), userID, tc.in)

			tc.expected(t, err)
		})
	}
}
