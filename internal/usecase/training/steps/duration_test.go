package steps_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/steps"
)

// callbackDuration90 and callbackDurationOther mirror steps' own private
// constants — black-box tests have to know the literal values.
const (
	callbackDuration90    = "training:duration:90"
	callbackDurationOther = "training:duration:other"
)

func TestDurationStep_Handle(t *testing.T) {
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
			name: "back goes to type step, duration untouched",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackBack},
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
				assert.Equal(t, model.TrainingDraftStepAwaitingType, d.Step)
				assert.Nil(t, d.DurationMinutes)
			},
		},

		{
			name: "stray callback is just acknowledged",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1"},
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
			name: "quick button saves duration and moves to notes",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackDuration90},
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
				assert.Equal(t, int32(90), *d.DurationMinutes)
				assert.Equal(t, model.TrainingDraftStepAwaitingNotes, d.Step)
			},
		},

		{
			name: "\"другое\" just nudges, doesn't set a duration",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackDurationOther},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					Send(chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.DurationMinutes)
			},
		},

		{
			name:    "no message, no callback — no-op",
			d:       &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unparsable duration is re-asked",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "ninety"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					Send(chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.DurationMinutes)
			},
		},

		{
			name: "zero duration is re-asked",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "0"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					Send(chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.DurationMinutes)
			},
		},

		{
			name: "valid duration moves to notes",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: " 90 "},
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
				assert.Equal(t, int32(90), *d.DurationMinutes)
				assert.Equal(t, model.TrainingDraftStepAwaitingNotes, d.Step)
			},
		},

		{
			name: "failed to persist duration",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "90"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
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

			step := steps.NewDuration(mockSender, mockRepo)

			err := step.Handle(context.Background(), tc.d, tc.in)

			tc.expected(t, tc.d, err)
		})
	}
}
