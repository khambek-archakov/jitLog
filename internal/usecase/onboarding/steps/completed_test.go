package steps_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/steps"
)

// callbackMenuSchedule mirrors steps' own private constant, same reasoning
// as callbackSkipAge/callbackBack in age_test.go. callbackMenuAddTraining
// is deliberately not tested here anymore — it's intercepted upstream by
// the router before CompletedStep ever sees it (see internal/usecase/router).
const callbackMenuSchedule = "menu:schedule"

func TestCompletedStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID int64 = 777

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(sender *Mocksender)
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "/start re-shows the menu",
			in:   dto.Input{ChatID: chatID, IsStartCmd: true},
			prepare: func(sender *Mocksender) {
				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "tapping a not-yet-built menu action shows a toast",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackMenuSchedule},
			prepare: func(sender *Mocksender) {
				sender.EXPECT().
					AnswerCallbackWithText("cb-1", gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(sender *Mocksender) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "random text re-shows the menu instead of being ignored",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "привет"},
			prepare: func(sender *Mocksender) {
				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:    "nothing to react to — no-op",
			in:      dto.Input{ChatID: chatID},
			prepare: func(sender *Mocksender) {},
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

			tc.prepare(mockSender)

			step := steps.NewCompleted(mockSender)

			err := step.Handle(context.Background(), &model.User{ID: 1}, tc.in)

			tc.expected(t, err)
		})
	}
}
