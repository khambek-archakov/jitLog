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
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/create/steps"
)

// These mirror steps' own private constants — black-box tests have to know
// the literal values.
const (
	callbackMenuAddTraining = "menu:add_training"
	callbackDateToday       = "training:date:today"
	callbackDateYesterday   = "training:date:yesterday"
	callbackDateOther       = "training:date:other"
	callbackDateCancel      = "training:date:cancel"
)

func TestDateStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID int64 = 777
	const messageID = 555

	tests := []struct {
		name    string
		d       *model.TrainingDraft
		in      dto.Input
		prepare func(
			sender *Mocksender,
			repo *MockdraftRepo,
		)
		expected func(t assert.TestingT, d *model.TrainingDraft, err error)
	}{
		{
			name: "kickoff shows the date keyboard",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackMenuAddTraining},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.Date)
			},
		},

		{
			name:    "no callback, no message — no-op",
			d:       &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "today button moves to type",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackDateToday},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, d.Date)
				assert.Equal(t, time.Now().Format("2006-01-02"), d.Date.Format("2006-01-02"))
				assert.Equal(t, model.TrainingDraftStepAwaitingType, d.Step)
			},
		},

		{
			name: "yesterday button moves to type",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackDateYesterday},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, d.Date)
				assert.Equal(t, time.Now().AddDate(0, 0, -1).Format("2006-01-02"), d.Date.Format("2006-01-02"))
				assert.Equal(t, model.TrainingDraftStepAwaitingType, d.Step)
			},
		},

		{
			name: "unparsable date text is re-asked",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "not a date"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					Send(chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.Date)
			},
		},

		{
			name: "day.month text is parsed",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "01.01"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, d.Date)
				assert.Equal(t, 1, d.Date.Day())
				assert.Equal(t, time.January, d.Date.Month())
				assert.Equal(t, model.TrainingDraftStepAwaitingType, d.Step)
			},
		},

		{
			name: "failed to persist date",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackDateToday},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "\"другая дата\" opens the calendar in place",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackDateOther},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, messageID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ int64, _ int, text string, keyboard dto.Keyboard) error {
						now := time.Now()
						assert.Contains(t, text, now.Format("2006"))

						// nav row, weekday row, N week rows, cancel row.
						require.GreaterOrEqual(t, len(keyboard), 4)
						require.Len(t, keyboard[0], 3)
						require.Len(t, keyboard[1], 7)

						last := keyboard[len(keyboard)-1]
						require.Len(t, last, 1)
						assert.Equal(t, "Отмена", last[0].Label)
						assert.Equal(t, callbackDateCancel, last[0].Data)

						// Can't page into the future from the current month.
						assert.Equal(t, "training:date:noop", keyboard[0][2].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "paging to a past month renders its full day grid",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:date:cal:2020-01"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, messageID, "📅 Январь 2020", gomock.Any()).
					DoAndReturn(func(_ int64, _ int, _ string, keyboard dto.Keyboard) error {
						// January 2020 is fully in the past, so every day 1-31
						// is a real, pickable button somewhere in the grid.
						found := map[string]bool{}
						for _, row := range keyboard {
							for _, b := range row {
								found[b.Data] = true
							}
						}

						assert.True(t, found["training:date:pick:2020-01-01"])
						assert.True(t, found["training:date:pick:2020-01-31"])
						// Paging further back must still be possible.
						assert.Equal(t, "training:date:cal:2019-12", keyboard[0][0].Data)
						// ...and forward, since 2020-01 isn't the current month.
						assert.Equal(t, "training:date:cal:2020-02", keyboard[0][2].Data)

						return nil
					})
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "malformed calendar month is just acknowledged",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:date:cal:not-a-month"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a calendar day moves to type",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:date:pick:2020-01-15"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				if assert.NotNil(t, d.Date) {
					assert.Equal(t, "2020-01-15", d.Date.Format("2006-01-02"))
				}
				assert.Equal(t, model.TrainingDraftStepAwaitingType, d.Step)
			},
		},

		{
			name: "malformed calendar day is just acknowledged",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:date:pick:not-a-date"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cancel returns to the quick date keyboard",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackDateCancel},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					EditMessageWithKeyboard(chatID, messageID, "Когда была тренировка?", gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
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

			err := step.Handle(context.Background(), tc.d, tc.in)

			tc.expected(t, tc.d, err)
		})
	}
}
