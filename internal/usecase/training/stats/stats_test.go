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
				repo.EXPECT().
					ListAllTrainings(gomock.Any(), userID).
					Return([]*model.Training{
						{Date: today, TrainingType: model.TrainingTypeGi, DurationMinutes: 60},
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
						assert.Contains(t, text, "Всё время")

						require.Len(t, kb, 2) // no mode-switch row
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
						require.Len(t, kb, 3) // period row + mode-switch row + back row

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

			err := uc.Handle(context.Background(), userID, tc.in)

			tc.expected(t, err)
		})
	}
}
