package stats_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/stats"
)

func TestUseCase_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	today := time.Now()

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name:    "no callback — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed period is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "stats:period:bogus"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to list trainings",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "stats:period:week"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "no trainings ever shows the empty state, belts never fetched",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "stats:period:week",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return(nil, nil)

				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Пока нет ни одной тренировки")

						require.Len(t, kb, 2)
						assert.Equal(t, "menu:add_training", kb[0][0].Data)
						assert.Equal(t, "menu:back", kb[1][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to list belt promotions",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "stats:period:week"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{{Date: today, DurationMinutes: 60}}, nil)

				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "shows stats for the requested period, mode switch hidden with one belt",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "stats:period:all",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				rounds := int16(6)

				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{
						{Date: today, TrainingType: model.TrainingTypeGi, DurationMinutes: 60, Rounds: &rounds},
						{Date: today, TrainingType: model.TrainingTypeNoGi, DurationMinutes: 90},
					}, nil)

				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return([]*model.BeltPromotion{{Belt: model.BeltWhite}}, nil)

				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Тренировок: 2")
						assert.Contains(t, text, "Схваток: 6")
						assert.Contains(t, text, "Всё время")

						require.Len(t, kb, 3) // period row + "other dates" row + back row (no mode-switch)
						assert.Equal(t, "• Всё время •", kb[0][3].Label)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "mode switch shown with two or more belts",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "stats:period:week",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{{Date: today, DurationMinutes: 60}}, nil)

				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return([]*model.BeltPromotion{{Belt: model.BeltWhite}, {Belt: model.BeltBlue}}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 4) // period row + "other dates" row + mode-switch row + back row

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "stats:belts with fewer than two belts falls back to the default period view",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "stats:belts",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return([]*model.BeltPromotion{{Belt: model.BeltWhite}}, nil)

				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{{Date: today, DurationMinutes: 60}}, nil)

				// handlePeriod re-fetches belts of its own accord.
				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return([]*model.BeltPromotion{{Belt: model.BeltWhite}}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "Неделя")

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "stats:belts shows the breakdown with two or more belts",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "stats:belts",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return([]*model.BeltPromotion{
						{Belt: model.BeltWhite, PromotedAt: today.AddDate(-1, 0, 0)},
						{Belt: model.BeltBlue, PromotedAt: today},
					}, nil)

				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{{Date: today, TrainingType: model.TrainingTypeGi, DurationMinutes: 60}}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "По поясам")
						assert.Equal(t, "• По поясам •", kb[0][1].Label)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "stats:belts with no trainings shows the empty state",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "stats:belts",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return([]*model.BeltPromotion{{Belt: model.BeltWhite}, {Belt: model.BeltBlue}}, nil)

				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return(nil, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "Пока нет ни одной тренировки")

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "stats:belts failed to list belt promotions",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "stats:belts"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "stats:belts failed to list trainings",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "stats:belts"},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return([]*model.BeltPromotion{{Belt: model.BeltWhite}, {Belt: model.BeltBlue}}, nil)

				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
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
			mockRepo := NewMocktrainingRepo(ctrl)
			mockBelts := NewMockbeltRepo(ctrl)

			tc.prepare(mockSender, mockRepo, mockBelts)

			uc := stats.New(mockSender, mockRepo, mockBelts)

			err := uc.Handle(context.Background(), &model.User{ID: userID}, tc.in)

			tc.expected(t, err)
		})
	}
}

// TestUseCase_Handle_Calendar covers the "📅 Другие даты" custom-range
// picker (cal:*) — a fixed clock throughout, since the calendar's own
// bounds (today, first available month) need to be deterministic.
func TestUseCase_Handle_Calendar(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	// A Tuesday — no edge-of-week/month surprises.
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }

	u := &model.User{ID: userID, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "cal:noop is acknowledged, screen untouched",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "cal:noop"},
			prepare: func(sender *Mocksender, _ *MocktrainingRepo, _ *MockbeltRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:x answers with the unavailable toast, screen untouched",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "cal:x"},
			prepare: func(sender *Mocksender, _ *MocktrainingRepo, _ *MockbeltRepo) {
				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Эта дата недоступна").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:c cancels back to the default week view",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "cal:c",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{{Date: now, DurationMinutes: 60}}, nil)

				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return([]*model.BeltPromotion{{Belt: model.BeltWhite}}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "Неделя")

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:f opens the От step at the given month",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:f:202610",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, _ *MockbeltRepo) {
				repo.EXPECT().ListAllTrainings(gomock.Any(), userID).Return(nil, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Equal(t, "Выбери начало периода", text)
						assert.Equal(t, "Отмена", kb[len(kb)-1][0].Label)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:d picks a valid От day, moves to the До step",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:d:20261015",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, _ *MockbeltRepo) {
				repo.EXPECT().ListAllTrainings(gomock.Any(), userID).Return(nil, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Equal(t, "Выбери конец периода\nНачало: 15 октября 2026", text)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:d for a stale/now-future day answers the unavailable toast instead",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:d:20261025",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, _ *MockbeltRepo) {
				repo.EXPECT().ListAllTrainings(gomock.Any(), userID).Return(nil, nil)

				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Эта дата недоступна").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:t navigates the До step to a new month",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:t:20261001:202609",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, _ *MockbeltRepo) {
				repo.EXPECT().ListAllTrainings(gomock.Any(), userID).Return(nil, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Equal(t, "Выбери конец периода\nНачало: 1 октября 2026", text)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:e picks a valid До day, shows the confirm screen",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:e:20261001:20261015",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, _ *MockbeltRepo) {
				repo.EXPECT().ListAllTrainings(gomock.Any(), userID).Return(nil, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Equal(t, "Период: 1 октября 2026 – 15 октября 2026", text)
						assert.Equal(t, "✅ Применить", kb[0][0].Label)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:e with До before От (stale tap) answers the unavailable toast",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:e:20261015:20261001",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, _ *MockbeltRepo) {
				repo.EXPECT().ListAllTrainings(gomock.Any(), userID).Return(nil, nil)

				sender.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", "Эта дата недоступна").Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:a applies the range and shows its stats",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:a:20261001:20261015",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, _ *MockbeltRepo) {
				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{
						{Date: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), TrainingType: model.TrainingTypeGi, DurationMinutes: 60},
					}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Contains(t, text, "Статистика · 1 октября 2026 – 15 октября 2026")
						assert.Contains(t, text, "Тренировок: 1")
						assert.NotContains(t, text, "Серия")
						assert.Equal(t, "📅 Другие даты", kb[1][0].Label)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:a with no trainings in range shows the empty-range text",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:a:20261001:20261015",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, _ *MockbeltRepo) {
				repo.EXPECT().ListAllTrainings(gomock.Any(), userID).Return(nil, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Contains(t, text, "В этом периоде нет тренировок.")

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
			mockRepo := NewMocktrainingRepo(ctrl)
			mockBelts := NewMockbeltRepo(ctrl)

			tc.prepare(mockSender, mockRepo, mockBelts)

			uc := stats.NewWithClock(mockSender, mockRepo, mockBelts, nowFn)

			err := uc.Handle(context.Background(), u, tc.in)

			tc.expected(t, err)
		})
	}
}

// TestUseCase_Handle_Calendar_Timezones guards the spec's own requirement:
// the calendar's "today" bound follows each user's own timezone, not the
// server's. Europe/Kaliningrad (UTC+2) and Asia/Yekaterinburg (UTC+5) are
// 3 hours apart — at this instant that's enough for their calendar days to
// actually differ.
func TestUseCase_Handle_Calendar_Timezones(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	now := time.Date(2026, 10, 20, 21, 30, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }

	kaliningrad := "Europe/Kaliningrad"
	yekaterinburg := "Asia/Yekaterinburg"

	tests := []struct {
		name        string
		tz          *string
		lastDayOf   string // the last pickable day of October, per that user's own "today".
		firstBeyond string // the first day after it — must be unavailable.
	}{
		{name: "Kaliningrad is still on the 20th", tz: &kaliningrad, lastDayOf: "20", firstBeyond: "21"},
		{name: "Yekaterinburg has already rolled over to the 21st", tz: &yekaterinburg, lastDayOf: "21", firstBeyond: "22"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			u := &model.User{
				ID: userID, Timezone: tc.tz, CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			}

			ctrl := gomock.NewController(t)

			mockSender := NewMocksender(ctrl)
			mockRepo := NewMocktrainingRepo(ctrl)
			mockBelts := NewMockbeltRepo(ctrl)

			mockRepo.EXPECT().ListAllTrainings(gomock.Any(), userID).Return(nil, nil)
			mockSender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

			mockSender.EXPECT().
				EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
					// October 2026 starts on a Thursday — 2 nav/header rows,
					// then 3 leading blanks before day 1 (see
					// TestCalDayRows_October2026).
					dayCell := func(day int) dto.Button {
						const leading = 3

						idx := leading + day - 1

						return kb[2+idx/7][idx%7]
					}

					last, _ := strconv.Atoi(tc.lastDayOf)
					beyond, _ := strconv.Atoi(tc.firstBeyond)

					assert.Equal(t, tc.lastDayOf, dayCell(last).Label, "day %d should be pickable", last)
					assert.NotEqual(t, "cal:x", dayCell(last).Data)

					assert.Equal(t, "·", dayCell(beyond).Label, "day %d should be unavailable", beyond)
					assert.Equal(t, "cal:x", dayCell(beyond).Data)

					return nil
				})

			uc := stats.NewWithClock(mockSender, mockRepo, mockBelts, nowFn)

			in := dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:f:202610",
			}

			err := uc.Handle(context.Background(), u, in)
			require.NoError(t, err)
		})
	}
}
