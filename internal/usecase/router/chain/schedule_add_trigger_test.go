package chain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestScheduleAddTrigger(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	t.Run("schedule:add begins a schedule draft", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:add"}

		err := run(t, u, in, func(m mocks) {
			noActiveDrafts(m, userID)

			m.scheduleCreate.EXPECT().
				Begin(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("any other callback skips to the next link", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:list"}

		err := run(t, u, in, func(m mocks) {
			noActiveDrafts(m, userID)

			// Reaching scheduleListTrigger is what proves this link skipped.
			m.scheduleList.EXPECT().
				Handle(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})
}
