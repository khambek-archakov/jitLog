package info_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/catalog/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, messageID int64 = 777, 555

	u := &model.User{ID: 42}

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockcatalogRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no callback — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed id is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:view:bogus"},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "not found",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:view:7"},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().GetCompetition(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Турнир не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:view:7"},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().GetCompetition(gomock.Any(), int64(7)).Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "shows the card with the primary source link",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:view:7",
			},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().GetCompetition(gomock.Any(), int64(7)).
					Return(&model.Competition{ID: 7, Title: "Moscow Open", Date: time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)}, nil)

				url := "https://example.com"
				repo.EXPECT().GetPrimarySourceURL(gomock.Any(), int64(7)).Return(&url, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Moscow Open")
						assert.Contains(t, text, "https://example.com")

						assert.Equal(t, "➕ В мои", kb[0][0].Label)
						assert.Equal(t, "catalog:add:7", kb[0][0].Data)
						assert.Equal(t, "← Назад", kb[1][0].Label)
						assert.Equal(t, "catalog:list", kb[1][0].Data)

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
			mockRepo := NewMockcatalogRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := info.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), u, tc.in)

			tc.expected(t, err)
		})
	}
}
