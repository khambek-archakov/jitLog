package delete_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	delete "github.com/khambek-archakov/jitLog/internal/usecase/competition/delete"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	u := &model.User{ID: userID}
	date := time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)

	baseCompetition := func() *model.UserCompetition {
		return &model.UserCompetition{ID: 7, UserID: userID, Title: "Moscow Open", Date: date}
	}

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
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:delete:bogus"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "not found shows notFoundText",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:delete:7"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Турнир не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "belongs to someone else shows notFoundText too",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:delete:7"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				c := baseCompetition()
				c.UserID = userID + 1
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(c, nil)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Турнир не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:delete:7"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "shows the confirm screen",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:delete:7",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "🗑 Удалить турнир?")
						assert.Contains(t, text, "Moscow Open")
						assert.Equal(t, "competition:delete:confirm:7", kb[0][0].Data)
						assert.Equal(t, "competition:view:7", kb[1][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed confirm id is just acknowledged",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:delete:confirm:bogus",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "confirm deletes and shows the deleted screen",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "competition:delete:confirm:7",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				repo.EXPECT().DeleteUserCompetition(gomock.Any(), int64(7)).Return(nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "🗑 Турнир удалён.", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						assert.Equal(t, "competition:list", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "confirm on something not found",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:delete:confirm:7",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Турнир не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to delete",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:delete:confirm:7",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				repo.EXPECT().DeleteUserCompetition(gomock.Any(), int64(7)).Return(errors.New("fail"))
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
			mockRepo := NewMockcompetitionRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := delete.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), u, tc.in)

			tc.expected(t, err)
		})
	}
}
