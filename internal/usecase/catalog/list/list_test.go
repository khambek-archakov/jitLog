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
	"github.com/khambek-archakov/jitLog/internal/usecase/catalog/list"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func nowDate(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	tests := []struct {
		name     string
		u        *model.User
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockcatalogRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no callback — no-op",
			u:       &model.User{ID: userID},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "no override — falls back to profile city",
			u:    &model.User{ID: userID, City: ptr("Москва")},
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:list",
			},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().GetViewFilter(gomock.Any(), userID).Return(nil, model.ErrNotFound)

				repo.EXPECT().
					ListCatalogUpcoming(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil()), 5, 0).
					DoAndReturn(func(_ context.Context, _ time.Time, city *string, _, _ int) ([]*model.Competition, bool, error) {
						require.NotNil(t, city)
						assert.Equal(t, "Москва", *city)

						return nil, false, nil
					})

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Москва")

						last := kb[len(kb)-1]
						assert.Equal(t, "← Назад", last[0].Label)
						assert.Equal(t, "competition:list", last[0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "explicit override (even nil) wins over profile city",
			u:    &model.User{ID: userID, City: ptr("Москва")},
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:list",
			},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().GetViewFilter(gomock.Any(), userID).Return(&model.CatalogViewFilter{UserID: userID, City: nil}, nil)

				repo.EXPECT().
					ListCatalogUpcoming(gomock.Any(), gomock.Any(), gomock.Nil(), 5, 0).
					Return(nil, false, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "rows, pagination and city controls render correctly",
			u:    &model.User{ID: userID},
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "catalog:list:page:1",
			},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().GetViewFilter(gomock.Any(), userID).Return(&model.CatalogViewFilter{UserID: userID, City: ptr("Казань")}, nil)

				city := "Казань"
				competitions := []*model.Competition{
					{ID: 7, Title: "Kazan Open", City: &city, Date: nowDate(2026, 11, 15)},
				}

				repo.EXPECT().ListCatalogUpcoming(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil()), 5, 5).Return(competitions, true, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), "🔎 Найти турнир", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 5)
						assert.Equal(t, "15 ноя — Kazan Open (Казань)", kb[0][0].Label)
						assert.Equal(t, "catalog:view:7", kb[0][0].Data)

						require.Len(t, kb[1], 2)
						assert.Equal(t, "‹", kb[1][0].Label)
						assert.Equal(t, "catalog:list:page:0", kb[1][0].Data)
						assert.Equal(t, "›", kb[1][1].Label)
						assert.Equal(t, "catalog:list:page:2", kb[1][1].Data)

						assert.Equal(t, "📍 Казань — изменить", kb[2][0].Label)
						assert.Equal(t, "catalog:city:prompt", kb[2][0].Data)

						assert.Equal(t, "Показать все города", kb[3][0].Label)
						assert.Equal(t, "catalog:city:all", kb[3][0].Data)

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
			name: "malformed page is just acknowledged",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:list:page:bogus"},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to list",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:list"},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().GetViewFilter(gomock.Any(), userID).Return(nil, model.ErrNotFound)
				repo.EXPECT().ListCatalogUpcoming(gomock.Any(), gomock.Any(), gomock.Any(), 5, 0).Return(nil, false, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "city prompt sets the draft and asks for a city, with a cancel button",
			u:    &model.User{ID: userID},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:city:prompt"},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().SetCityDraft(gomock.Any(), userID).Return(nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 1)
						assert.Equal(t, "❌ Отмена", kb[0][0].Label)
						assert.Equal(t, "catalog:city:cancel", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cancelling the city prompt clears the draft and shows the list again",
			u:    &model.User{ID: userID},
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:city:cancel",
			},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().DeleteCityDraft(gomock.Any(), userID).Return(nil)
				repo.EXPECT().GetViewFilter(gomock.Any(), userID).Return(nil, model.ErrNotFound)
				repo.EXPECT().ListCatalogUpcoming(gomock.Any(), gomock.Any(), gomock.Any(), 5, 0).Return(nil, false, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "show all cities sets an explicit nil override",
			u:    &model.User{ID: userID},
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:city:all",
			},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().SetViewFilter(gomock.Any(), userID, (*string)(nil)).Return(nil)
				repo.EXPECT().GetViewFilter(gomock.Any(), userID).Return(&model.CatalogViewFilter{UserID: userID}, nil)
				repo.EXPECT().ListCatalogUpcoming(gomock.Any(), gomock.Any(), gomock.Nil(), 5, 0).Return(nil, false, nil)
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)
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

			uc := list.New(mockSender, mockRepo)

			err := uc.Handle(context.Background(), tc.u, tc.in)

			tc.expected(t, err)
		})
	}
}

func TestUseCase_Continue(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	u := &model.User{ID: userID}

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockcatalogRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no message — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "blank city is re-asked",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "   "},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "too-long city is re-asked",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: stringOfLen(81)},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				sender.EXPECT().Send(gomock.Any(), chatID, gomock.Any()).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "valid city sets the filter and sends a fresh list",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: " Казань "},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().
					SetViewFilter(gomock.Any(), userID, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, city *string) error {
						require.NotNil(t, city)
						assert.Equal(t, "Казань", *city)

						return nil
					})

				repo.EXPECT().DeleteCityDraft(gomock.Any(), userID).Return(nil)

				repo.EXPECT().
					ListCatalogUpcoming(gomock.Any(), gomock.Any(), gomock.Not(gomock.Nil()), 5, 0).
					Return(nil, false, nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "Казань")

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to persist",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "Казань"},
			prepare: func(sender *Mocksender, repo *MockcatalogRepo) {
				repo.EXPECT().SetViewFilter(gomock.Any(), userID, gomock.Any()).Return(errors.New("fail"))
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
			mockRepo := NewMockcatalogRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := list.New(mockSender, mockRepo)

			err := uc.Continue(context.Background(), u, &model.CatalogCityDraft{UserID: userID}, tc.in)

			tc.expected(t, err)
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func stringOfLen(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = 'a'
	}

	return string(b)
}
