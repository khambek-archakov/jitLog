package info_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	u := &model.User{ID: userID}

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockcompetitionRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no callback — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed id is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:view:bogus"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "not found",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:view:7"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Турнир не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "belongs to someone else",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:view:7"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).
					Return(&model.UserCompetition{ID: 7, UserID: userID + 1}, nil)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Турнир не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:view:7"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "shows the card",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "competition:view:7",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).
					Return(&model.UserCompetition{ID: 7, UserID: userID, Title: "Moscow Open"}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "Moscow Open")

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
			mockRepo := NewMockcompetitionRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := info.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), u, tc.in)

			tc.expected(t, err)
		})
	}
}
