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

// These mirror TimeStep's own private constants — black-box tests have to
// know the literal values.
const (
	callbackBack      = "schedule:draft:back"
	callbackTimeOther = "schedule:draft:time:other"
)

func TestTimeStep_Handle(t *testing.T) {
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
			name: "back returns to the day question",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingTime},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackBack},
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
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.ScheduleDraftStepAwaitingDay, d.Step)
			},
		},

		{
			name: "failed to persist back",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingTime},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackBack},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(errors.New("fail"))
			},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "other nudges for free text",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingTime},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackTimeOther},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)

				sender.EXPECT().
					Send(gomock.Any(), chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "preset pick saves time and moves to type",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingTime},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:time:preset:1900"},
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
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
				require.NotNil(t, d.TimeMinutes)
				assert.Equal(t, int16(19*60), *d.TimeMinutes)
				assert.Equal(t, model.ScheduleDraftStepAwaitingType, d.Step)
			},
		},

		{
			name: "malformed preset token is just acknowledged",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingTime},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:draft:time:preset:bogus"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)
			},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.TimeMinutes)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingTime},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					AnswerCallback(gomock.Any(), "cb-1").
					Return(nil)
			},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:    "no callback, no message — no-op",
			d:       &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingTime},
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "free text HH:MM saves time and moves to type",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingTime},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "18:30"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				repo.EXPECT().
					UpdateDraft(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
				require.NotNil(t, d.TimeMinutes)
				assert.Equal(t, int16(18*60+30), *d.TimeMinutes)
				assert.Equal(t, model.ScheduleDraftStepAwaitingType, d.Step)
			},
		},

		{
			name: "unparseable free text is rejected",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingTime},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "not a time"},
			prepare: func(sender *Mocksender, repo *MockdraftRepo) {
				sender.EXPECT().
					Send(gomock.Any(), chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t *testing.T, d *model.ScheduleDraft, err error) {
				assert.NoError(t, err)
				assert.Nil(t, d.TimeMinutes)
			},
		},

		{
			name: "failed to persist free text time",
			d:    &model.ScheduleDraft{ID: 1, Step: model.ScheduleDraftStepAwaitingTime},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "08:00"},
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

			step := steps.NewTime(mockSender, mockRepo)

			err := step.Handle(context.Background(), tc.d, tc.in)

			tc.expected(t, tc.d, err)
		})
	}
}
