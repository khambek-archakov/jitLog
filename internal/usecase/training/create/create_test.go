package create_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/create"
)

var trainingDate = time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

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
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
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

			uc := create.New(mockSender, mockRepo)

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
			d: &model.TrainingDraft{
				ID: 1, UserID: 42, Date: &trainingDate, TrainingType: model.TrainingTypeGi,
				Step: model.TrainingDraftStepAwaitingDuration,
			},
			in: dto.Input{ChatID: chatID, HasMessage: true, Text: "60"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateTraining(gomock.Any(), int64(42), trainingDate, model.TrainingTypeGi, int32(60), gomock.Nil()).
					Return(&model.Training{ID: 7, Date: trainingDate, TrainingType: model.TrainingTypeGi, DurationMinutes: 60}, nil)

				repo.EXPECT().
					DeleteDraft(gomock.Any(), int64(42)).
					Return(nil)

				confirmation := sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, "Вот что я умею:", gomock.Any()).
					After(confirmation).
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

		{
			name: "cancel deletes the draft and shows the main menu, regardless of step",
			d:    &model.TrainingDraft{ID: 1, UserID: 42, Step: model.TrainingDraftStepAwaitingDuration},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:cancel"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					DeleteDraft(gomock.Any(), int64(42)).
					Return(nil)

				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, "Вот что я умею:", gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to delete draft on cancel",
			d:    &model.TrainingDraft{ID: 1, UserID: 42, Step: model.TrainingDraftStepAwaitingDate},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:cancel"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					DeleteDraft(gomock.Any(), int64(42)).
					Return(errors.New("fail"))
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

			uc := create.New(mockSender, mockRepo)

			err := uc.Continue(context.Background(), tc.d, tc.in)

			tc.expected(t, err)
		})
	}
}
