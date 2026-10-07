package add_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/catalog/add"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	u := &model.User{ID: userID}
	date := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *Mockrepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no callback — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *Mockrepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed id is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:add:bogus"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "not found",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:add:7"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				repo.EXPECT().GetCompetition(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Соревнование не найдено.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "copies the catalog entry into the personal list",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:add:7"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				city := "Москва"
				repo.EXPECT().GetCompetition(gomock.Any(), int64(7)).
					Return(&model.Competition{ID: 7, Title: "Moscow Open", Date: date, City: &city}, nil)

				url := "https://example.com"
				repo.EXPECT().GetPrimarySourceURL(gomock.Any(), int64(7)).Return(&url, nil)

				repo.EXPECT().
					CreateUserCompetitionFromCatalog(gomock.Any(), userID, int64(7), "Moscow Open", date, (*time.Time)(nil), &city, &url).
					Return(&model.UserCompetition{ID: 99, UserID: userID, CompetitionID: ptrInt64(7), Title: "Moscow Open", Date: date, City: &city, URL: &url}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "✅ Добавлено в твои соревнования")
						assert.Contains(t, text, "Moscow Open")

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to create",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:add:7"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				repo.EXPECT().GetCompetition(gomock.Any(), int64(7)).
					Return(&model.Competition{ID: 7, Title: "Moscow Open", Date: date}, nil)

				repo.EXPECT().GetPrimarySourceURL(gomock.Any(), int64(7)).Return(nil, nil)

				repo.EXPECT().
					CreateUserCompetitionFromCatalog(gomock.Any(), userID, int64(7), "Moscow Open", date, (*time.Time)(nil), (*string)(nil), (*string)(nil)).
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
			mockRepo := NewMockrepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := add.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), u, tc.in)

			tc.expected(t, err)
		})
	}
}

func ptrInt64(v int64) *int64 {
	return &v
}
