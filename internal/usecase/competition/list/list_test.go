package list_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/list"
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
			name: "unknown callback is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "empty list shows the empty state with just add/back",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "competition:list",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().ListUpcoming(gomock.Any(), userID, gomock.Any()).Return(nil, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Пока нет предстоящих турниров")

						require.Len(t, kb, 2)
						assert.Equal(t, "➕ Добавить турнир", kb[0][0].Label)
						assert.Equal(t, "← Главное меню", kb[1][0].Label)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "non-empty list shows rows plus add/past/back",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "competition:list",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				city := "Москва"
				competitions := []*model.UserCompetition{
					{ID: 7, Title: "Moscow Open", City: &city, Date: nowDate(2026, 11, 15)},
					{ID: 8, Title: "No City Cup", Date: nowDate(2026, 12, 1)},
				}

				repo.EXPECT().ListUpcoming(gomock.Any(), userID, gomock.Any()).Return(competitions, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "🏆 Соревнования", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 5)
						assert.Equal(t, "15 ноя — Moscow Open (Москва)", kb[0][0].Label)
						assert.Equal(t, "competition:view:7", kb[0][0].Data)
						assert.Equal(t, "1 дек — No City Cup", kb[1][0].Label)
						assert.Equal(t, "➕ Добавить турнир", kb[2][0].Label)
						assert.Equal(t, "Прошедшие", kb[3][0].Label)
						assert.Equal(t, "competition:history:page:0", kb[3][0].Data)
						assert.Equal(t, "← Главное меню", kb[4][0].Label)

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
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:list",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().ListUpcoming(gomock.Any(), userID, gomock.Any()).Return(nil, errors.New("fail"))
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

			uc := list.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), u, tc.in)

			tc.expected(t, err)
		})
	}
}
