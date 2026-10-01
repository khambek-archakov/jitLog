package chain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestScheduleDraftContinuation(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	// Every case here is already past draftContinuation and
	// editDraftContinuation.
	noOtherDrafts := func(m mocks) {
		m.drafts.EXPECT().
			GetDraftByUserID(gomock.Any(), userID).
			Return(nil, model.ErrNotFound)

		m.edits.EXPECT().
			GetEditDraftByUserID(gomock.Any(), userID).
			Return(nil, model.ErrNotFound)
	}

	t.Run("active schedule draft delegates to create.Continue", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "19:00"}
		draft := &model.ScheduleDraft{UserID: userID}

		err := run(t, u, in, func(m mocks) {
			noOtherDrafts(m)

			m.scheduleDrafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(draft, nil)

			m.scheduleCreate.EXPECT().
				Continue(gomock.Any(), draft, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("no active schedule draft skips to the next link", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "menu:add_training"}

		err := run(t, u, in, func(m mocks) {
			noOtherDrafts(m)

			m.scheduleDrafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			m.scheduleEdits.EXPECT().
				GetEditDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			// Reaching addTrainingTrigger is what proves this link skipped.
			m.create.EXPECT().
				Begin(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("repo error propagates and stops the chain", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "hi"}

		err := run(t, u, in, func(m mocks) {
			noOtherDrafts(m)

			m.scheduleDrafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(nil, errors.New("fail"))
		})

		assert.Error(t, err)
	})
}
