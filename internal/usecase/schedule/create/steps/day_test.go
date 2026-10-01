package steps_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/create/steps"
)

// callbackScheduleAdd mirrors DayStep's own private constant — black-box
// tests have to know the literal value to trigger the kickoff branch.
const callbackScheduleAdd = "schedule:add"

func TestDayStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID int64 = 777

	tests := []struct {
		name     string
		d        *model.ScheduleDraft
		in       dto.Input
		prepare  func(sender *Mocksender, repo *MockdraftRepo)
		expected func(t *testing.T, d *model.ScheduleDraft, err error)
	}{
		{
			name: "kickoff shows the day keyboard",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingDay},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackScheduleAdd},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ int64, _ string, keyboard dto.Keyboard) error {
						require.Len(t, keyboard, 2) // 7 weekdays in one row + cancel row
						require.Len(t, keyboard[0], 7)
						assert.Equal(t, "Пн", keyboard[0][0].Label)
						assert.Equal(t, "schedule:draft:day:1", keyboard[0][0].Data)
						assert.Equal(t, "Вс", keyboard[0][6].Label)
						assert.Equal(t, "schedule:draft:day:7", keyboard[0][6].Data)
						assert.Equal(t, "❌ Отмена", keyboard[1][0].Label)

						return nil
					})
			},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.DayOfWeek)
			},
		},

		{
			name:    "no callback — no-op",
			d:       &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingDay},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingDay},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)
			},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a day moves to time",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingDay},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:day:3"},
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
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
				require.NotNil(t, d.DayOfWeek)
				assert.Equal(t, int16(3), *d.DayOfWeek)
				assert.Equal(t, model.ScheduleDraftStepAwaitingTime, d.Step)
			},
		},

		{
			name: "out of range day is just acknowledged",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingDay},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:day:8"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)
			},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.DayOfWeek)
			},
		},

		{
			name: "failed to persist day",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingDay},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:day:1"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(errors.New("fail"))
			},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
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

			step := steps.NewDay(mockSender, mockRepo)

			err := step.Handle(context.Background(), tc.d, tc.in)

			tc.expected(t, tc.d, err)
		})
	}
}
