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
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/history"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func nowDate(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

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
			name: "malformed page is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:history:page:bogus"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "empty page",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "competition:history:page:0",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().ListPast(gomock.Any(), userID, gomock.Any(), 5, 0).Return(nil, false, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Прошедших турниров пока нет")

						require.Len(t, kb, 1)
						assert.Equal(t, "← Назад", kb[0][0].Label)
						assert.Equal(t, "competition:list", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "page with rows and a next page",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "competition:history:page:1",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				result := "2nd place"
				competitions := []*model.UserCompetition{
					{ID: 7, Title: "Moscow Open", Date: nowDate(2026, 9, 1), Result: &result},
					{ID: 8, Title: "No Result Cup", Date: nowDate(2026, 8, 1)},
				}

				repo.EXPECT().ListPast(gomock.Any(), userID, gomock.Any(), 5, 5).Return(competitions, true, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "Прошедшие турниры", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 4)
						assert.Equal(t, "1 сен — Moscow Open · 2nd place", kb[0][0].Label)
						assert.Equal(t, "competition:view:7", kb[0][0].Data)
						assert.Equal(t, "1 авг — No Result Cup", kb[1][0].Label)

						require.Len(t, kb[2], 2)
						assert.Equal(t, "‹", kb[2][0].Label)
						assert.Equal(t, "competition:history:page:0", kb[2][0].Data)
						assert.Equal(t, "›", kb[2][1].Label)
						assert.Equal(t, "competition:history:page:2", kb[2][1].Data)

						assert.Equal(t, "← Назад", kb[3][0].Label)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to list",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:history:page:0",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().ListPast(gomock.Any(), userID, gomock.Any(), 5, 0).Return(nil, false, errors.New("fail"))
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

			uc := history.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), u, tc.in)

			tc.expected(t, err)
		})
	}
}
