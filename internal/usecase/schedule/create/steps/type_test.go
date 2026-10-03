package steps_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/create/steps"
)

func TestTypeStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID, userID int64 = 777, 42

	day, minutes := int16(3), int16(19*60)

	draft := func() *model.ScheduleDraft {
		return &model.ScheduleDraft{
			ID: 1, UserID: userID, Step: model.ScheduleDraftStepAwaitingType,
			DayOfWeek: &day, TimeMinutes: &minutes,
		}
	}

	tests := []struct {
		name     string
		d        *model.ScheduleDraft
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockdraftRepo)
		expected func(t *testing.T, err error)
	}{
		{
			name: "back returns to the time question",
			d:    draft(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:back"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to persist back",
			d:    draft(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:back"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(errors.New("fail"))
			},
			expected: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "picking Gi creates the slot, deletes the draft and shows confirmation",
			d:    draft(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:type:gi"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateSlot(gomock.Any(), userID, day, minutes, model.TrainingTypeGi).
					Return(&model.ScheduleSlot{
						ID: 5, UserID: userID, DayOfWeek: day, TimeMinutes: minutes, TrainingType: model.TrainingTypeGi,
					}, nil)

				repo.EXPECT().
					DeleteDraft(gomock.Any(), userID).
					Return(nil)

				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, "✅ Добавлено: Ср — 19:00, 🥋 Gi", gomock.Any()).
					Return(nil)
			},
			expected: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking No-Gi creates the slot",
			d:    draft(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:type:no_gi"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateSlot(gomock.Any(), userID, day, minutes, model.TrainingTypeNoGi).
					Return(&model.ScheduleSlot{
						ID: 5, UserID: userID, DayOfWeek: day, TimeMinutes: minutes, TrainingType: model.TrainingTypeNoGi,
					}, nil)

				repo.EXPECT().
					DeleteDraft(gomock.Any(), userID).
					Return(nil)

				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking Open Mat creates the slot",
			d:    draft(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:type:open_mat"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateSlot(gomock.Any(), userID, day, minutes, model.TrainingTypeOpenMat).
					Return(&model.ScheduleSlot{
						ID: 5, UserID: userID, DayOfWeek: day, TimeMinutes: minutes, TrainingType: model.TrainingTypeOpenMat,
					}, nil)

				repo.EXPECT().
					DeleteDraft(gomock.Any(), userID).
					Return(nil)

				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			d:    draft(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)
			},
			expected: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:    "no callback — no-op",
			d:       draft(),
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:    "missing day on draft is a programmer error",
			d:       &model.ScheduleDraft{ID: 1, UserID: userID, Step: model.ScheduleDraftStepAwaitingType, TimeMinutes: &minutes},
			in:      dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:type:gi"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},

		{
			name:    "missing time on draft is a programmer error",
			d:       &model.ScheduleDraft{ID: 1, UserID: userID, Step: model.ScheduleDraftStepAwaitingType, DayOfWeek: &day},
			in:      dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:type:gi"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "failed to create slot",
			d:    draft(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:type:gi"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateSlot(gomock.Any(), userID, day, minutes, model.TrainingTypeGi).
					Return(nil, errors.New("fail"))
			},
			expected: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "failed to delete draft after creating slot",
			d:    draft(),
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:type:gi"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					CreateSlot(gomock.Any(), userID, day, minutes, model.TrainingTypeGi).
					Return(&model.ScheduleSlot{
						ID: 5, UserID: userID, DayOfWeek: day, TimeMinutes: minutes, TrainingType: model.TrainingTypeGi,
					}, nil)

				repo.EXPECT().
					DeleteDraft(gomock.Any(), userID).
					Return(errors.New("fail"))
			},
			expected: func(t *testing.T, err error) {
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

			tc.expected(t, err)
		})
	}
}
