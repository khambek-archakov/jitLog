package chain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestProfileTrigger(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	t.Run("profile:* delegates to profile.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "profile:show"}

		err := run(t, u, in, func(m mocks) {
			noActiveDrafts(m, userID)

			m.profile.EXPECT().
				Handle(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("any other callback skips to the fallback", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"}

		err := run(t, u, in, func(m mocks) {
			noActiveDrafts(m, userID)

			// Reaching onboarding (as the chain's last link) is what proves
			// this link skipped — profileTrigger is the last trigger before
			// it in the handler list.
			m.onboarding.EXPECT().
				Handle(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})
}
