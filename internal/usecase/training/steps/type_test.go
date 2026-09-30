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

// callbackTrainingTypeGi and callbackBack mirror steps' own private
// constants — black-box tests have to know the literal values.
const (
	callbackTrainingTypeGi = "training:type:gi"
	callbackBack           = "training:back"
)

func TestTypeStep_Handle(t *testing.T) {
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
			name:    "no callback — no-op",
			d:       &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingType},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t assert.TestingT, d *model.TrainingDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingType},
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
			name: "picking a type moves to duration",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingType},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackTrainingTypeGi},
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
				assert.Equal(t, model.TrainingTypeGi, d.TrainingType)
				assert.Equal(t, model.TrainingDraftStepAwaitingDuration, d.Step)
			},
		},

		{
			name: "back goes to date step, type untouched",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingType},
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
				assert.Equal(t, model.TrainingDraftStepAwaitingDate, d.Step)
				assert.Equal(t, model.TrainingTypeNone, d.TrainingType)
			},
		},

		{
			name: "failed to persist type",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingType},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackTrainingTypeGi},
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

			step := steps.NewType(mockSender, mockRepo)

			err := step.Handle(context.Background(), tc.d, tc.in)

			tc.expected(t, tc.d, err)
		})
	}
}
