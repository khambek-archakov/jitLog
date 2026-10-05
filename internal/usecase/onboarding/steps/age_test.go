package steps_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/steps"
)

// AgeStep no longer asks anything — the age question was removed from
// onboarding. It only exists so a user whose onboarding_step was already
// stuck at awaiting_age before this shipped gets silently forwarded to the
// belt step on their very next message, regardless of what that input is.
func TestAgeStep_Handle(t *testing.T) {
	t.Parallel()

	const chatID int64 = 777

	name := "test"

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(user *Mockuser, sender *Mocksender)
		expected func(t assert.TestingT, u *model.User, err error)
	}{
		{
			name: "/start forwards straight to the belt question",
			in:   dto.Input{ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.OnboardingStepAwaitingBelt, u.OnboardingStep)
			},
		},

		{
			name: "any other input also forwards to the belt question",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "25"},
			prepare: func(user *Mockuser, sender *Mocksender) {
				user.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(nil)

				sender.EXPECT().
					SendWithKeyboard(gomock.Any(), chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, u *model.User, err error) {
				assert.NoError(t, err)
				assert.Equal(t, model.OnboardingStepAwaitingBelt, u.OnboardingStep)
			},
		},

		{
			name: "failed to persist step",
			in:   dto.Input{ChatID: chatID, IsStartCmd: true},
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

			u := &model.User{ID: 1, Name: &name}
			err := step.Handle(context.Background(), u, tc.in)

			tc.expected(t, u, err)
		})
	}
}
