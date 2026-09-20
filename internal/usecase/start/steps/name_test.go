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

func TestNameStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID int64 = 777

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
			name: "/start greets",
			u:    &model.User{ID: 1},
			in:   dto.Input{ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser, sender *Mocksender) {
				sender.EXPECT().
					Send(chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "stray callback is just acknowledged",
			u:    &model.User{ID: 1},
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1"},
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
			name: "blank name is re-asked",
			u:    &model.User{ID: 1},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "   "},
			prepare: func(user *Mockuser, sender *Mocksender) {
				sender.EXPECT().
					Send(chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "failed to persist name",
			u:    &model.User{ID: 1},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "test"},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(errors.New("fail"))
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "valid name moves to age step",
			u:    &model.User{ID: 1},
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "  test  "},
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
				assert.Equal(t, "test", *u.Name)
				assert.Equal(t, model.OnboardingStepAwaitingAge, u.OnboardingStep)
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

			step := steps.NewName(mockSender, mockUser)

			err := step.Handle(context.Background(), tc.u, tc.in)

			tc.expected(t, tc.u, err)
		})
	}
}
