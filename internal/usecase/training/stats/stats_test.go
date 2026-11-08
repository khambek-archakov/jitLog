package stats_test

import (
	"context"
	"errors"
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
			name: "share button shows on the Month tab once it has a training",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "stats:period:month",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{{Date: today, DurationMinutes: 60}}, nil)

				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return([]*model.BeltPromotion{{Belt: model.BeltWhite}}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 4) // period row + "other dates" row + share row + back row
						assert.Equal(t, "📤 Поделиться", kb[2][0].Label)
						assert.Equal(t, "share:month", kb[2][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "share button hidden on the Month tab when that month has no trainings",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "stats:period:month",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				// Only a training from well outside the current month —
				// len(all) != 0, so the empty-state screen never fires,
				// but filterByPeriod(month) still comes back empty.
				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{{Date: today.AddDate(-1, 0, 0), DurationMinutes: 60}}, nil)

				belts.EXPECT().
					ListBeltPromotions(gomock.Any(), userID).
					Return([]*model.BeltPromotion{{Belt: model.BeltWhite}}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, _ string, kb dto.Keyboard) error {
						require.Len(t, kb, 3) // period row + "other dates" row + back row, no share row
						for _, row := range kb {
							assert.NotEqual(t, "share:month", row[0].Data)
						}

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
// picker (cal:*). Every date is pickable now (see calendar.go's own
// doc) — there's no "unavailable day" response left to test here, only
// the ordinary step transitions, the От/До swap, and the bold-month
// MarkdownV2 screens.
func TestUseCase_Handle_Calendar(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	u := &model.User{ID: userID}

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
			name: "cal:c cancels back to the default week view",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "cal:c",
			},
			prepare: func(sender *Mocksender, repo *MocktrainingRepo, belts *MockbeltRepo) {
				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{{Date: time.Now(), DurationMinutes: 60}}, nil)

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
			name: "cal:f opens the От step at the given month, bolded in MarkdownV2",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:f:202610",
			},
			prepare: func(sender *Mocksender, _ *MocktrainingRepo, _ *MockbeltRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboardMarkdown(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Equal(t, "Выбери начало периода\n*Октябрь 2026*", text)

						require.Len(t, kb[0], 4) // «, ‹, ›, » — no label button.
						assert.Equal(t, "Отмена", kb[len(kb)-1][0].Label)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:d picks any От day — a far-future date included — and moves to the До step",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:d:20991015",
			},
			prepare: func(sender *Mocksender, _ *MocktrainingRepo, _ *MockbeltRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboardMarkdown(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Equal(t, "Выбери конец периода\nНачало: 15 октября 2099\n*Октябрь 2099*", text)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:d picking a day from an adjacent month opens До on that day's own month",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:d:20260831", // tapped while viewing September's grid.
			},
			prepare: func(sender *Mocksender, _ *MocktrainingRepo, _ *MockbeltRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboardMarkdown(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Equal(t, "Выбери конец периода\nНачало: 31 августа 2026\n*Август 2026*", text)

						return nil
					})
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
			prepare: func(sender *Mocksender, _ *MocktrainingRepo, _ *MockbeltRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboardMarkdown(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Equal(t, "Выбери конец периода\nНачало: 1 октября 2026\n*Сентябрь 2026*", text)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:e with До after От shows the confirm screen",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:e:20261001:20261015",
			},
			prepare: func(sender *Mocksender, _ *MocktrainingRepo, _ *MockbeltRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Equal(t, "Период: 1 октября 2026 – 15 октября 2026", text)
						assert.Equal(t, "✅ Применить", kb[0][0].Label)
						assert.Equal(t, "cal:a:20261001:20261015", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:e with До before От is swapped instead of rejected",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:e:20261015:20261001",
			},
			prepare: func(sender *Mocksender, _ *MocktrainingRepo, _ *MockbeltRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, kb dto.Keyboard) error {
						assert.Equal(t, "Период: 1 октября 2026 – 15 октября 2026", text)
						assert.Equal(t, "cal:a:20261001:20261015", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cal:e with equal От and До is a valid single-day period",
			in: dto.Input{
				ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1",
				CallbackData: "cal:e:20261001:20261001",
			},
			prepare: func(sender *Mocksender, _ *MocktrainingRepo, _ *MockbeltRepo) {
				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
						assert.Equal(t, "Период: 1 октября 2026 – 1 октября 2026", text)

						return nil
					})
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

			uc := stats.New(mockSender, mockRepo, mockBelts)

			err := uc.Handle(context.Background(), u, tc.in)

			tc.expected(t, err)
		})
	}
}

// TestUseCase_Handle_Share covers the "📤 Поделиться" monthly recap image
// (share:month) — a fixed clock, since "this month" needs to be
// deterministic.
func TestUseCase_Handle_Share(t *testing.T) {
	t.Parallel()

	const chatID, userID, messageID int64 = 777, 42, 555

	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	nowFn := func() time.Time { return now }

	name := "Алексей"
	u := &model.User{ID: userID, Name: &name}

	tests := []struct {
		name     string
		prepare  func(sender *Mocksender, repo *MocktrainingRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "renders and sends the month's recap as a new photo message",
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				rounds := int16(21)

				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{
						{Date: time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC), TrainingType: model.TrainingTypeGi, DurationMinutes: 60, Rounds: &rounds},
						{Date: time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), TrainingType: model.TrainingTypeGi, DurationMinutes: 60}, // outside October — must not count.
					}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().SendUploadPhotoAction(gomock.Any(), chatID).Return(nil)

				sender.EXPECT().
					SendPhotoWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, photo []byte, filename, caption string, kb dto.Keyboard) error {
						assert.NotEmpty(t, photo)
						assert.Equal(t, "jitlog-month.png", filename)
						assert.Contains(t, caption, "Поделись своим прогрессом")
						assert.Contains(t, caption, "src_share")
						assert.Equal(t, "← Главное меню", kb[0][0].Label)
						assert.Equal(t, "menu:back", kb[0][0].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "a stale tap with nothing left in the month sends nothing",
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().ListAllTrainings(gomock.Any(), userID).Return(nil, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().SendUploadPhotoAction(gomock.Any(), chatID).Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "a chat-action failure doesn't block the photo",
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{{Date: now, DurationMinutes: 60}}, nil)

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().SendUploadPhotoAction(gomock.Any(), chatID).Return(errors.New("fail"))
				sender.EXPECT().
					SendPhotoWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to list trainings",
			prepare: func(sender *Mocksender, repo *MocktrainingRepo) {
				repo.EXPECT().ListAllTrainings(gomock.Any(), userID).Return(nil, errors.New("fail"))

				sender.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
				sender.EXPECT().SendUploadPhotoAction(gomock.Any(), chatID).Return(nil)
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

			tc.prepare(mockSender, mockRepo)

			uc := stats.NewWithClock(mockSender, mockRepo, mockBelts, nowFn)

			in := dto.Input{ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "share:month"}

			err := uc.Handle(context.Background(), u, in)

			tc.expected(t, err)
		})
	}
}
