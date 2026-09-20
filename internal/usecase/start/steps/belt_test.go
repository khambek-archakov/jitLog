package steps_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/steps"
)

// callbackBeltWhite mirrors steps' own private constant, same reasoning as
// callbackSkipAge/callbackBack in age_test.go.
const callbackBeltWhite = "start:belt:white"

func TestBeltStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID int64 = 777

	name := "test"

	tests := []struct {
		name    string
		u       *model.User
		in      dto.Input
		prepare func(
			user *Mockuser,
			sender *Mocksender,
		)
		expected func(t assert.TestingT, u *model.User, err error)
	}{
		{
			name: "/start re-asks the belt question",
			u:    &model.User{ID: 1, Name: &name},
			in:   dto.Input{ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser, sender *Mocksender) {
				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:    "no callback — no-op",
			u:       &model.User{ID: 1, Name: &name},
			in:      dto.Input{ChatID: chatID},
			prepare: func(user *Mockuser, sender *Mocksender) {},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "back goes to the age step, belt untouched",
			u:    &model.User{ID: 1, Name: &name},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackBack},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.OnboardingStepAwaitingAge, u.OnboardingStep)
				assert.Equal(t, model.BeltNone, u.Belt)
			},
		},

		{
			name: "unknown callback is just acknowledged",
			u:    &model.User{ID: 1, Name: &name},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
			prepare: func(user *Mockuser, sender *Mocksender) {
				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "picking a belt completes onboarding",
			u:    &model.User{ID: 1, Name: &name},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackBeltWhite},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					AnswerCallback("cb-1").
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.BeltWhite, u.Belt)
				assert.Equal(t, model.OnboardingStepCompleted, u.OnboardingStep)
			},
		},

		{
			name: "failed to persist belt",
			u:    &model.User{ID: 1, Name: &name},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackBeltWhite},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockUser := NewMockuser(ctrl)
			mockSender := NewMocksender(ctrl)

			tc.prepare(mockUser, mockSender)

			step := steps.NewBelt(mockSender, mockUser)

			err := step.Handle(context.Background(), tc.u, tc.in)

			tc.expected(t, tc.u, err)
		})
	}
}
