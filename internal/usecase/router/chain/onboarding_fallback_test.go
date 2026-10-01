package chain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestOnboardingFallback(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	tests := []struct {
		name string
		in   dto.Input
	}{
		{
			name: "plain message nobody else claimed",
			in:   dto.Input{ChatID: chatID, HasMessage: true, Text: "привет"},
		},
		{
			name: "callback nobody else claimed",
			in:   dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := run(t, u, tc.in, func(m mocks) {
				noActiveDrafts(m, userID)

				m.onboarding.EXPECT().
					Handle(gomock.Any(), u, tc.in).
					Return(nil)
			})

			assert.NoError(t, err)
		})
	}
}
