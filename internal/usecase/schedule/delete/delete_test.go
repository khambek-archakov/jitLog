package delete_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	delete "github.com/khambek-archakov/jitLog/internal/usecase/schedule/delete"
)

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	slot := &model.ScheduleSlot{ID: 7, UserID: userID, DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi}

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
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:delete:nope"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				sender.EXPECT().AnswerCallback("cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "slot not found",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:delete:7"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				sender.EXPECT().AnswerCallbackWithText("cb-1", "Слот не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "slot belongs to someone else",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:delete:7"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				other := *slot
				other.UserID = userID + 1
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(&other, nil)
				sender.EXPECT().AnswerCallbackWithText("cb-1", "Слот не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch slot",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:delete:7"},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "shows a confirm screen",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:delete:7",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Удалить слот?")
						assert.Equal(t, "schedule:delete:confirm:7", kb[0][0].Data)
						assert.Equal(t, "schedule:view:7", kb[1][0].Data)

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
				CallbackData: "schedule:delete:confirm:7",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				repo.EXPECT().DeleteSlot(gomock.Any(), int64(7)).Return(nil)

				sender.EXPECT().AnswerCallback("cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, int(messageID), "🗑 Слот удалён.", gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 1)
						assert.Equal(t, "schedule:list", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "confirm on someone else's slot is refused",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:delete:confirm:7",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				other := *slot
				other.UserID = userID + 1
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(&other, nil)
				sender.EXPECT().AnswerCallbackWithText("cb-1", "Слот не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to delete slot",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:delete:confirm:7",
			},
			prepare: func(sender *Mocksender, repo *MockslotRepo) {
				repo.EXPECT().GetSlot(gomock.Any(), int64(7)).Return(slot, nil)
				repo.EXPECT().DeleteSlot(gomock.Any(), int64(7)).Return(errors.New("fail"))
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

			uc := delete.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), userID, tc.in)

			tc.expected(t, err)
		})
	}
}
