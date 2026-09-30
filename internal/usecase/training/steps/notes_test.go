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

const callbackSkipNotes = "training:notes:skip"

func TestNotesStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	duration := int32(60)
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		d       *model.TrainingDraft
		in      dto.Input
		prepare func(
			sender *Mocksender,
			repo *MockdraftRepo,
		)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "skip finishes without notes",
			d: &model.TrainingDraft{
				ID: 1, UserID: userID, Date: &date, TrainingType: model.TrainingTypeGi,
				DurationMinutes: &duration, Step: model.TrainingDraftStepAwaitingNotes,
			},
			in: dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackSkipNotes},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				repo.EXPECT().
					CreateTraining(gomock.Any(), userID, date, model.TrainingTypeGi, int32(60), gomock.Nil()).
					Return(&model.Training{ID: 1}, nil)

				repo.EXPECT().
					DeleteDraft(gomock.Any(), userID).
					Return(nil)

				sender.EXPECT().
					Send(chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			d:    &model.TrainingDraft{ID: 1, UserID: userID, Step: model.TrainingDraftStepAwaitingNotes},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:    "no message, no callback — no-op",
			d:       &model.TrainingDraft{ID: 1, UserID: userID, Step: model.TrainingDraftStepAwaitingNotes},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "text note finishes with notes saved",
			d: &model.TrainingDraft{
				ID: 1, UserID: userID, Date: &date, TrainingType: model.TrainingTypeNoGi,
				DurationMinutes: &duration, Step: model.TrainingDraftStepAwaitingNotes,
			},
			in: dto.Input{ChatID: chatID, HasMessage: true, Text: "  armbar from closed guard  "},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateTraining(gomock.Any(), userID, date, model.TrainingTypeNoGi, int32(60), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ int64, _ time.Time, _ model.TrainingType, _ int32, notes *string) (*model.Training, error) {
						assert.NotNil(t, notes)
						assert.Equal(t, "armbar from closed guard", *notes)

						return &model.Training{ID: 1}, nil
					})

				repo.EXPECT().
					DeleteDraft(gomock.Any(), userID).
					Return(nil)

				sender.EXPECT().
					Send(chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to create training",
			d: &model.TrainingDraft{
				ID: 1, UserID: userID, Date: &date, TrainingType: model.TrainingTypeGi,
				DurationMinutes: &duration, Step: model.TrainingDraftStepAwaitingNotes,
			},
			in: dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackSkipNotes},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				repo.EXPECT().
					CreateTraining(gomock.Any(), userID, date, model.TrainingTypeGi, int32(60), gomock.Nil()).
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
			mockRepo := NewMockdraftRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			step := steps.NewNotes(mockSender, mockRepo)

			err := step.Handle(context.Background(), tc.d, tc.in)

			tc.expected(t, err)
		})
	}
}
