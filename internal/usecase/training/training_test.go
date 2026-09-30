package training_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training"
)

const callbackMenuAddTraining = "menu:add_training"

func TestUseCase_Begin(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackMenuAddTraining}

	tests := []struct {
		name    string
		prepare func(
			sender *Mocksender,
			repo *MockdraftRepo,
		)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "failed to create draft",
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateDraft(gomock.Any(), userID).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "creates the draft and shows the date question",
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateDraft(gomock.Any(), userID).
					Return(&model.TrainingDraft{ID: 1, UserID: userID, Step: model.TrainingDraftStepAwaitingDate}, nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
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
			mockRepo := NewMockdraftRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := training.New(mockSender, mockRepo)

			err := uc.Begin(context.Background(), userID, in)

			tc.expected(t, err)
		})
	}
}

func TestUseCase_Continue(t *testing.T) {
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
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "dispatches to the step matching the draft's current step",
			d:    &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "60"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:    "unknown step is a no-op",
			d:       &model.TrainingDraft{ID: 1, Step: model.TrainingDraftStep("bogus")},
			in:      dto.Input{ChatID: chatID, HasMessage: true, Text: "60"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
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
			mockRepo := NewMockdraftRepo(ctrl)

			tc.prepare(mockSender, mockRepo)

			uc := training.New(mockSender, mockRepo)

			err := uc.Continue(context.Background(), tc.d, tc.in)

			tc.expected(t, err)
		})
	}
}
