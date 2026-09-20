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

// callbackSkipAge and callbackBack mirror steps' own private constants —
// they're part of the bot's wire contract with itself, not exported, so
// black-box tests have to know the literal values.
const (
	callbackSkipAge = "start:age:skip"
	callbackBack    = "start:back"
)

func TestAgeStep_Handle(t *testing.T) {
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
			name: "/start re-asks the age question",
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
			name:    "no message, no callback — no-op",
			u:       &model.User{ID: 1, Name: &name},
			in:      dto.Input{ChatID: chatID},
			prepare: func(user *Mockuser, sender *Mocksender) {},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "back goes to the name step, age untouched",
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
					Send(chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.OnboardingStepAwaitingName, u.OnboardingStep)
				assert.Nil(t, u.Age)
			},
		},

		{
			name: "skip moves straight to belt, age stays empty",
			u:    &model.User{ID: 1, Name: &name},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: callbackSkipAge},
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
				assert.Equal(t, model.OnboardingStepAwaitingBelt, u.OnboardingStep)
				assert.Nil(t, u.Age)
			},
		},

		{
			name: "stray callback is just acknowledged",
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
			name: "unparsable age is skipped with a note",
			u:    &model.User{ID: 1, Name: &name},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "twenty five"},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					Send(chatID, gomock.Any()).
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.OnboardingStepAwaitingBelt, u.OnboardingStep)
				assert.Nil(t, u.Age)
			},
		},

		{
			name: "valid age is stored",
			u:    &model.User{ID: 1, Name: &name},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "25"},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.OnboardingStepAwaitingBelt, u.OnboardingStep)
				assert.Equal(t, int16(25), *u.Age)
			},
		},

		{
			name: "failed to persist age",
			u:    &model.User{ID: 1, Name: &name},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "25"},
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

			step := steps.NewAge(mockSender, mockUser)

			err := step.Handle(context.Background(), tc.u, tc.in)

			tc.expected(t, tc.u, err)
		})
	}
}
