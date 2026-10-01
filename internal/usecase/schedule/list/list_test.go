package list_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/list"
)

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42
	const messageID = 555

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
			name: "schedule:list shows the list",
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:list"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().
					ListSlots(gomock.Any(), userID).
					Return([]*model.ScheduleSlot{
						{DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi},
					}, nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, messageID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to list slots",
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:list"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().
					ListSlots(gomock.Any(), userID).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)
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
			mockRepo := NewMockslotRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := list.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), userID, tc.in)

			tc.expected(t, err)
		})
	}
}
