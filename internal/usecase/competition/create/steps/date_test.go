package steps_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/create/steps"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestDateStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	title := "Moscow Open"
	u := &model.User{ID: userID}

	tests := []struct {
		name     string
		d        *model.UserCompetitionDraft
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockdraftRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "stray callback is just acknowledged",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Title: &title},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "back goes to the title step, title untouched",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Title: &title},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:draft:back"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, d *model.UserCompetitionDraft) error {
						assert.Equal(t, model.UserCompetitionDraftStepAwaitingTitle, d.Step)
						assert.Equal(t, "Moscow Open", *d.Title)

						return nil
					})

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "Как называется турнир?")

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to persist on back",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Title: &title},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:draft:back"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().UpdateDraft(gomock.Any(), gomock.Any()).Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name:    "no message, no callback — no-op",
			d:       &model.UserCompetitionDraft{ID: 1, UserID: userID, Title: &title},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unparsable date is re-asked",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Title: &title},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "not a date"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "valid date with year finishes the wizard",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Title: &title},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "20.11.2026"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				want := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

				repo.EXPECT().
					CreateUserCompetition(gomock.Any(), userID, "Moscow Open", want).
					Return(&model.UserCompetition{ID: 7, Title: "Moscow Open", Date: want}, nil)

				repo.EXPECT().DeleteDraft(gomock.Any(), userID).Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "✅ Турнир добавлен")
						assert.Contains(t, text, "Moscow Open")

						require.Len(t, kb, 5)
						assert.Equal(t, "➕ Город", kb[0][0].Label)
						assert.Equal(t, "competition:edit:7:city", kb[0][0].Data)
						assert.Equal(t, "➕ Ссылка", kb[0][1].Label)
						assert.Equal(t, "competition:edit:7:url", kb[0][1].Data)
						assert.Equal(t, "➕ Дата окончания", kb[1][0].Label)
						assert.Equal(t, "competition:edit:7:end_date", kb[1][0].Data)
						assert.Equal(t, "✏️ Изменить", kb[2][0].Label)
						assert.Equal(t, "competition:edit:7", kb[2][0].Data)
						assert.Equal(t, "🗑️ Удалить", kb[3][0].Label)
						assert.Equal(t, "competition:delete:7", kb[3][0].Data)
						assert.Equal(t, "← Назад", kb[4][0].Label)
						assert.Equal(t, "competition:list", kb[4][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "valid date without year assumes current year",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Title: &title},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "20.11"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				want := time.Date(time.Now().Year(), 11, 20, 0, 0, 0, 0, time.UTC)

				repo.EXPECT().
					CreateUserCompetition(gomock.Any(), userID, "Moscow Open", want).
					Return(&model.UserCompetition{ID: 7, Title: "Moscow Open", Date: want}, nil)

				repo.EXPECT().DeleteDraft(gomock.Any(), userID).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to create",
			d:    &model.UserCompetitionDraft{ID: 1, UserID: userID, Title: &title},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "20.11.2026"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateUserCompetition(gomock.Any(), userID, "Moscow Open", gomock.Any()).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name:    "no title set defensively errors instead of panicking",
			d:       &model.UserCompetitionDraft{ID: 1, UserID: userID},
			in:      dto.Input{ChatID: chatID, HasMessage: true, Text: "20.11.2026"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
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
			mockRepo := NewMockdraftRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			step := steps.NewDate(mockSender, mockRepo)

			err := step.Handle(context.Background(), u, tc.d, tc.in)

			tc.expected(t, err)
		})
	}
}
