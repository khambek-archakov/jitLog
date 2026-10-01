package chain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestOnboardingGate(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	t.Run("onboarding not complete delegates to onboarding", func(t *testing.T) {
		t.Parallel()

		u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepAwaitingAge}
		in := dto.Input{ChatID: chatID, IsStartCmd: true}

		err := run(t, u, in, func(m mocks) {
			m.onboarding.EXPECT().
				Handle(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("onboarding complete skips to the next link", func(t *testing.T) {
		t.Parallel()

		u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}
		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "hi"}

		err := run(t, u, in, func(m mocks) {
			// Reaching draftContinuation's own lookup is what proves the
			// gate didn't intercept a completed user.
			m.drafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(&model.TrainingDraft{ID: 1, UserID: userID}, nil)

			m.create.EXPECT().
				Continue(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)
		})

		assert.NoError(t, err)
	})
}
