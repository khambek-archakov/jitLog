package steps_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/steps"
)

// callbackMenuAddTraining, callbackDateToday and callbackDateYesterday
// mirror steps' own private constants — black-box tests have to know the
// literal values.
const (
	callbackMenuAddTraining = "menu:add_training"
	callbackDateToday       = "training:date:today"
	callbackDateYesterday   = "training:date:yesterday"
)

func TestDateStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID int64 = 777

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
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
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

			err := step.Handle(context.Background(), tc.d, tc.in)

			tc.expected(t, tc.d, err)
		})
	}
}
