package info_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/info"
)

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockslotRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no callback — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed id is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:view:bogus"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "shows the card",
			in:   dto.Input{ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:view:5"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().
					GetSlot(gomock.Any(), int64(5)).
					Return(&model.ScheduleSlot{
						ID: 5, UserID: userID, DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi,
					}, nil)

				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "not found slot shows a toast",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:view:5"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().
					GetSlot(gomock.Any(), int64(5)).
					Return(nil, model.ErrNotFound)

				sender.EXPECT().
					AnswerCallbackWithText(gomock.Any(), "cb-1", "Слот не найден.").
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "slot belonging to another user shows a toast",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:view:5"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().
					GetSlot(gomock.Any(), int64(5)).
					Return(&model.ScheduleSlot{ID: 5, UserID: 999}, nil)

				sender.EXPECT().
					AnswerCallbackWithText(gomock.Any(), "cb-1", "Слот не найден.").
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to get slot",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:view:5"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().
					GetSlot(gomock.Any(), int64(5)).
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

			mockSender := NewMocksender(ctrl)
			mockRepo := NewMockslotRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := info.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), userID, tc.in)

			tc.expected(t, err)
		})
	}
}
