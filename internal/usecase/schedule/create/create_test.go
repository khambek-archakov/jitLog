package create_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/create"
)

const callbackScheduleAdd = "schedule:add"

func TestUseCase_Begin(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackScheduleAdd}

	tests := []struct {
		name     string
		prepare  func(sender *Mocksender, repo *MockdraftRepo)
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
			name: "creates the draft and shows the day question",
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateDraft(gomock.Any(), userID).
					Return(&model.ScheduleDraft{ID: 1, UserID: userID, Step: model.ScheduleDraftStepAwaitingDay}, nil)

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
		name     string
		d        *model.ScheduleDraft
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockdraftRepo)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "dispatches to the step matching the draft's current step",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingType},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:back"},
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
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:    "unknown step is a no-op",
			d:       &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStep("bogus")},
			in:      dto.Input{ChatID: chatID, HasCallback: true},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "cancel deletes the draft and shows the main menu, regardless of step",
			d:    &model.ScheduleDraft{ID: 1, UserID: 42, Step: model.ScheduleDraftStepAwaitingTime},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:cancel"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					DeleteDraft(gomock.Any(), int64(42)).
					Return(nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, "Вот что я умею:", gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to delete draft on cancel",
			d:    &model.ScheduleDraft{ID: 1, UserID: 42, Step: model.ScheduleDraftStepAwaitingDay},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:cancel"},
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
