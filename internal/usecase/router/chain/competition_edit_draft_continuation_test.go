package chain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestCompetitionEditDraftContinuation(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	// Every case here is already past every other continuation, including
	// competitionDraftContinuation — this is the last link before the
	// callback-prefix triggers.
	noOtherDrafts := func(m mocks) {
		m.drafts.EXPECT().
			GetDraftByUserID(gomock.Any(), userID).
			Return(nil, model.ErrNotFound)

		m.edits.EXPECT().
			GetEditDraftByUserID(gomock.Any(), userID).
			Return(nil, model.ErrNotFound)

		m.scheduleDrafts.EXPECT().
			GetDraftByUserID(gomock.Any(), userID).
			Return(nil, model.ErrNotFound)

		m.scheduleEdits.EXPECT().
			GetEditDraftByUserID(gomock.Any(), userID).
			Return(nil, model.ErrNotFound)

		m.profileEdits.EXPECT().
			GetEditDraftByUserID(gomock.Any(), userID).
			Return(nil, model.ErrNotFound)

		m.competitionDrafts.EXPECT().
			GetDraftByUserID(gomock.Any(), userID).
			Return(nil, model.ErrNotFound)
	}

	t.Run("active competition edit draft delegates to update.Continue", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "Moscow"}
		draft := &model.UserCompetitionEditDraft{UserID: userID, UserCompetitionID: 7}

		err := run(t, u, in, func(m mocks) {
			noOtherDrafts(m)

			m.competitionEdits.EXPECT().
				GetEditDraftByUserID(gomock.Any(), userID).
				Return(draft, nil)

			m.competitionUpdate.EXPECT().
				Continue(gomock.Any(), u, draft, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("no active competition edit draft skips to the next link", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "menu:add_training"}

		err := run(t, u, in, func(m mocks) {
			noOtherDrafts(m)

			m.competitionEdits.EXPECT().
				GetEditDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			// Reaching training's own addTrigger is what proves this link
			// skipped — competitionEditDraftContinuation is the last
			// continuation before training's sub-package.
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

			m.competitionEdits.EXPECT().
				GetEditDraftByUserID(gomock.Any(), userID).
				Return(nil, errors.New("fail"))
		})

		assert.Error(t, err)
	})
}
