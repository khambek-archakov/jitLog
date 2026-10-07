package update_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/update"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const chatID, userID, messageID int64 = 777, 42, 555

var date = time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)

func baseUser() *model.User {
	return &model.User{ID: userID}
}

func baseCompetition() *model.UserCompetition {
	return &model.UserCompetition{ID: 7, UserID: userID, Title: "Moscow Open", Date: date}
}

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

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
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:edit:bogus"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "not found",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:edit:7"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Соревнование не найдено.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "belongs to someone else",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:edit:7"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				c := baseCompetition()
				c.UserID = userID + 1
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(c, nil)
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Соревнование не найдено.").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to fetch",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:edit:7"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "bare id shows the edit menu",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:edit:7",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "✏️ Что изменить?", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 7)
						assert.Equal(t, "competition:edit:7:title", kb[0][0].Data)
						assert.Equal(t, "Название", kb[0][0].Label)
						assert.Equal(t, "competition:edit:7:date", kb[1][0].Data)
						assert.Equal(t, "Дата", kb[1][0].Label)
						assert.Equal(t, "competition:edit:7:end_date", kb[2][0].Data)
						assert.Equal(t, "➕ Дата окончания", kb[2][0].Label)
						assert.Equal(t, "competition:edit:7:city", kb[3][0].Data)
						assert.Equal(t, "➕ Город", kb[3][0].Label)
						assert.Equal(t, "competition:edit:7:url", kb[4][0].Data)
						assert.Equal(t, "➕ Ссылка", kb[4][0].Label)
						assert.Equal(t, "competition:edit:7:result", kb[5][0].Data)
						assert.Equal(t, "➕ Результат", kb[5][0].Label)
						assert.Equal(t, "competition:view:7", kb[6][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "bare id shows the edit menu — filled optional fields drop the ➕",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:edit:7",
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				city, url, result := "Москва", "https://example.com", "2nd place"
				c := baseCompetition()
				c.City, c.URL, c.Result, c.EndDate = &city, &url, &result, &date

				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(c, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "✏️ Что изменить?", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						assert.Equal(t, "Дата окончания", kb[2][0].Label)
						assert.Equal(t, "Город", kb[3][0].Label)
						assert.Equal(t, "Ссылка", kb[4][0].Label)
						assert.Equal(t, "Результат", kb[5][0].Label)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown action is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:edit:7:bogus"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, field := range []string{"title", "date", "end_date", "city", "url", "result"} {
		editField := map[string]model.UserCompetitionEditField{
			"title":    model.UserCompetitionEditFieldTitle,
			"date":     model.UserCompetitionEditFieldDate,
			"end_date": model.UserCompetitionEditFieldEndDate,
			"city":     model.UserCompetitionEditFieldCity,
			"url":      model.UserCompetitionEditFieldURL,
			"result":   model.UserCompetitionEditFieldResult,
		}[field]

		tests = append(tests, struct {
			name     string
			in       dto.Input
			prepare  func(sender *Mocksender, repo *MockcompetitionRepo)
			expected func(t assert.TestingT, err error)
		}{
			name: field + " sets a pending edit draft and prompts",
			in: dto.Input{
				ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:edit:7:" + field,
			},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				repo.EXPECT().SetEditDraft(gomock.Any(), userID, int64(7), editField).Return(nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ string, kb dto.Keyboard) error {
						assert.Equal(t, "competition:edit:7:cancel", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		})
	}

	tests = append(tests, struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockcompetitionRepo)
		expected func(t assert.TestingT, err error)
	}{
		name: "cancel clears the draft and shows the card again",
		in: dto.Input{
			ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
			CallbackData: "competition:edit:7:cancel",
		},
		prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
			repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
			repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)

			sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

			sender.EXPECT().
				EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
				Return(nil)
		},
		expected: func(t assert.TestingT, err error) {
			assert.NoError(t, err)
		},
	})

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockSender := NewMocksender(ctrl)
			mockRepo := NewMockcompetitionRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := update.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), baseUser(), tc.in)

			tc.expected(t, err)
		})
	}
}

func TestUseCase_Continue(t *testing.T) {
	t.Parallel()

	draft := func(field model.UserCompetitionEditField) *model.UserCompetitionEditDraft {
		return &model.UserCompetitionEditDraft{UserID: userID, UserCompetitionID: 7, Field: field}
	}

	tests := []struct {
		name     string
		d        *model.UserCompetitionEditDraft
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockcompetitionRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no message — no-op",
			d:       draft(model.UserCompetitionEditFieldTitle),
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "competition gone meanwhile just clears the draft",
			d:    draft(model.UserCompetitionEditFieldTitle),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "new title"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(nil, model.ErrNotFound)
				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "empty title is re-asked",
			d:    draft(model.UserCompetitionEditFieldTitle),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "   "},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "valid title finishes the edit",
			d:    draft(model.UserCompetitionEditFieldTitle),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: " Worlds "},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)

				repo.EXPECT().
					UpdateUserCompetition(gomock.Any(), int64(7), "Worlds", date, gomock.Nil(), gomock.Nil(), gomock.Nil(), gomock.Nil()).
					Return(&model.UserCompetition{ID: 7, Title: "Worlds", Date: date}, nil)

				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unparsable date is re-asked",
			d:    draft(model.UserCompetitionEditFieldDate),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "not a date"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "new start date after existing end date is re-asked",
			d:    draft(model.UserCompetitionEditFieldDate),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "20.11.2026"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				c := baseCompetition()
				endDate := date.AddDate(0, 0, 2)
				c.EndDate = &endDate
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(c, nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "valid date finishes the edit",
			d:    draft(model.UserCompetitionEditFieldDate),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "20.11.2026"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)

				newDate := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)

				repo.EXPECT().
					UpdateUserCompetition(gomock.Any(), int64(7), "Moscow Open", newDate, gomock.Nil(), gomock.Nil(), gomock.Nil(), gomock.Nil()).
					Return(&model.UserCompetition{ID: 7, Title: "Moscow Open", Date: newDate}, nil)

				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "end date before start date is re-asked",
			d:    draft(model.UserCompetitionEditFieldEndDate),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "01.11.2026"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "valid end date finishes the edit",
			d:    draft(model.UserCompetitionEditFieldEndDate),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "17.11.2026"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)

				endDate := time.Date(2026, 11, 17, 0, 0, 0, 0, time.UTC)

				repo.EXPECT().
					UpdateUserCompetition(gomock.Any(), int64(7), "Moscow Open", date, &endDate, gomock.Nil(), gomock.Nil(), gomock.Nil()).
					Return(&model.UserCompetition{ID: 7, Title: "Moscow Open", Date: date, EndDate: &endDate}, nil)

				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "too-long city is re-asked",
			d:    draft(model.UserCompetitionEditFieldCity),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: stringOfLen(81)},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "blank city clears it",
			d:    draft(model.UserCompetitionEditFieldCity),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "   "},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)

				repo.EXPECT().
					UpdateUserCompetition(gomock.Any(), int64(7), "Moscow Open", date, gomock.Nil(), gomock.Nil(), gomock.Nil(), gomock.Nil()).
					Return(&model.UserCompetition{ID: 7, Title: "Moscow Open", Date: date}, nil)

				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "valid city finishes the edit",
			d:    draft(model.UserCompetitionEditFieldCity),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: " Москва "},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)

				repo.EXPECT().
					UpdateUserCompetition(gomock.Any(), int64(7), "Moscow Open", date, gomock.Nil(), gomock.Any(), gomock.Nil(), gomock.Nil()).
					DoAndReturn(func(
						_ context.Context, _ int64, _ string, _ time.Time, _ *time.Time, city, _, _ *string,
					) (*model.UserCompetition, error) {
						require.NotNil(t, city)
						assert.Equal(t, "Москва", *city)

						return &model.UserCompetition{ID: 7, Title: "Moscow Open", Date: date, City: city}, nil
					})

				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "url without scheme is re-asked",
			d:    draft(model.UserCompetitionEditFieldURL),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "example.com"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "valid url is normalized and finishes the edit",
			d:    draft(model.UserCompetitionEditFieldURL),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "https://EXAMPLE.com/page?utm_source=x"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)

				repo.EXPECT().
					UpdateUserCompetition(gomock.Any(), int64(7), "Moscow Open", date, gomock.Nil(), gomock.Nil(), gomock.Any(), gomock.Nil()).
					DoAndReturn(func(
						_ context.Context, _ int64, _ string, _ time.Time, _ *time.Time, _, url, _ *string,
					) (*model.UserCompetition, error) {
						require.NotNil(t, url)
						assert.Equal(t, "https://example.com/page", *url)

						return &model.UserCompetition{ID: 7, Title: "Moscow Open", Date: date, URL: url}, nil
					})

				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "too-long result is re-asked",
			d:    draft(model.UserCompetitionEditFieldResult),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: stringOfLen(201)},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "valid result finishes the edit",
			d:    draft(model.UserCompetitionEditFieldResult),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "2nd place"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)

				repo.EXPECT().
					UpdateUserCompetition(gomock.Any(), int64(7), "Moscow Open", date, gomock.Nil(), gomock.Nil(), gomock.Nil(), gomock.Any()).
					Return(&model.UserCompetition{ID: 7, Title: "Moscow Open", Date: date}, nil)

				repo.EXPECT().DeleteEditDraft(gomock.Any(), userID).Return(nil)
				sender.EXPECT().SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to update",
			d:    draft(model.UserCompetitionEditFieldTitle),
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "Worlds"},
			prepare: func(sender *Mocksender, repo *MockcompetitionRepo) {
				repo.EXPECT().GetUserCompetition(gomock.Any(), int64(7)).Return(baseCompetition(), nil)

				repo.EXPECT().
					UpdateUserCompetition(gomock.Any(), int64(7), "Worlds", date, gomock.Nil(), gomock.Nil(), gomock.Nil(), gomock.Nil()).
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
			mockRepo := NewMockcompetitionRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := update.New(mockSender, mockRepo)

			err := uc.Continue(context.Background(), baseUser(), tc.d, tc.in)

			tc.expected(t, err)
		})
	}
}

func stringOfLen(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = 'a'
	}

	return string(b)
}
