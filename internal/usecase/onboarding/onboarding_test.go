package onboarding_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
)

func TestUseCase_Handle(t *testing.T) {
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
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "new user is greeted on /start",
			u:    &model.User{ID: 1, OnboardingStep: model.OnboardingStepAwaitingName},
			in:   dto.Input{ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser, sender *Mocksender) {
				sender.EXPECT().
					Send(chatID, gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "user mid-onboarding is re-asked their pending question on /start",
			u:    &model.User{ID: 2, Name: &name, OnboardingStep: model.OnboardingStepAwaitingAge},
			in:   dto.Input{ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser, sender *Mocksender) {
				sender.EXPECT().
					SendWithKeyboard(chatID, gomock.Any(), gomock.Any()).
					Return(nil)
			},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name:    "unknown onboarding step — no handler, no-op",
			u:       &model.User{ID: 3, OnboardingStep: model.OnboardingStep("bogus")},
			in:      dto.Input{ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser, sender *Mocksender) {},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
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

			uc := onboarding.New(mockSender, mockUser)

			err := uc.Handle(context.Background(), tc.u, tc.in)

			tc.expected(t, err)
		})
	}
}
