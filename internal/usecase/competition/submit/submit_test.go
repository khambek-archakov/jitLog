package submit_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/submit"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const adminChatID int64 = 999

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	u := &model.User{ID: userID}
	date := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	url := "https://example.com/moscow-open"

	withURL := func() *model.UserCompetition {
		return &model.UserCompetition{ID: 7, UserID: userID, Title: "Moscow Open", Date: date, URL: &url}
	}

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
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:submit:bogus"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "not found",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:submit:7"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Турнир не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "belongs to someone else",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:submit:7"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				c := withURL()
				c.UserID = userID + 1
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(c, nil)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Турнир не найден.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "no url is rejected",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:submit:7"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).
					Return(&model.UserCompetition{ID: 7, UserID: userID, Date: date}, nil)
				sender.EXPECT().
					AnswerCallbackWithText(gomock.Any(), "cb-1", "Нужна ссылка, чтобы предложить турнир в каталог.").
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "exact url match attaches silently",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:submit:7"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(withURL(), nil)
				repo.EXPECT().FindSourceByURL(gomock.Any(), url).Return(&model.CompetitionSource{CompetitionID: 50, URL: url}, nil)
				repo.EXPECT().LinkUserCompetition(gomock.Any(), int64(7), int64(50)).Return(nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "no url match, no city — creates new directly and notifies admin",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:submit:7"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(withURL(), nil)
				repo.EXPECT().FindSourceByURL(gomock.Any(), url).Return(nil, model.ErrNotFound)

				repo.EXPECT().CreateCompetition(gomock.Any(), "Moscow Open", date, (*time.Time)(nil), (*string)(nil), userID).
					Return(&model.Competition{ID: 55, Title: "Moscow Open", Date: date}, nil)
				repo.EXPECT().AddCompetitionSource(gomock.Any(), int64(55), url, true).Return(nil)
				repo.EXPECT().LinkUserCompetition(gomock.Any(), int64(7), int64(55)).Return(nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), adminChatID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Moscow Open")
						assert.Contains(t, text, url)
						assert.Equal(t, "✅ Одобрить", kb[0][0].Label)
						assert.Equal(t, "competition:moderate:55:approve", kb[0][0].Data)
						assert.Equal(t, "❌ Отклонить", kb[0][1].Label)
						assert.Equal(t, "competition:moderate:55:reject", kb[0][1].Data)
						assert.Equal(t, "🔗 Это дубль", kb[1][0].Label)
						assert.Equal(t, "competition:moderate:55:dup", kb[1][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "candidates found — shows the picker",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:submit:7"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				city := "Москва"
				c := withURL()
				c.City = &city

				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(c, nil)
				repo.EXPECT().FindSourceByURL(gomock.Any(), url).Return(nil, model.ErrNotFound)

				candidates := []*model.Competition{
					{ID: 60, Title: "Moscow Open 2026", Date: date, City: &city},
				}
				repo.EXPECT().FindCandidates(gomock.Any(), date, "Москва").Return(candidates, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, "Это один из них?", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 2)
						assert.Equal(t, "competition:submit:7:attach:60", kb[0][0].Data)
						assert.Equal(t, "Нет, это новый турнир", kb[1][0].Label)
						assert.Equal(t, "competition:submit:7:new", kb[1][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a candidate attaches as non-primary",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:submit:7:attach:60",
			},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(withURL(), nil)
				repo.EXPECT().AddCompetitionSource(gomock.Any(), int64(60), url, false).Return(nil)
				repo.EXPECT().LinkUserCompetition(gomock.Any(), int64(7), int64(60)).Return(nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "declining all candidates creates a new entry",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:submit:7:new"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(withURL(), nil)
				repo.EXPECT().CreateCompetition(gomock.Any(), "Moscow Open", date, (*time.Time)(nil), (*string)(nil), userID).
					Return(&model.Competition{ID: 55, Title: "Moscow Open", Date: date}, nil)
				repo.EXPECT().AddCompetitionSource(gomock.Any(), int64(55), url, true).Return(nil)
				repo.EXPECT().LinkUserCompetition(gomock.Any(), int64(7), int64(55)).Return(nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), adminChatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to find source",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:submit:7"},
			prepare: func(sender *Mocksender, repo *Mockrepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(withURL(), nil)
				repo.EXPECT().FindSourceByURL(gomock.Any(), url).Return(nil, errors.New("fail"))
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

			uc := submit.New(mockSender, mockRepo, adminChatID)

			err := uc.Handle(context.Background(), u, tc.in)

			tc.expected(t, err)
		})
	}
}
