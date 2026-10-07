package chain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestCompetitionDraftContinuation(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	// Every case here is already past draftContinuation, editDraftContinuation,
	// scheduleDraftContinuation, scheduleEditDraftContinuation and
	// profileEditDraftContinuation.
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
	}

	t.Run("active competition draft delegates to create.Continue", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "20.11.2026"}
		draft := &model.UserCompetitionDraft{UserID: userID}

		err := run(t, u, in, func(m mocks) {
			noOtherDrafts(m)

			m.competitionDrafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(draft, nil)

			m.competitionCreate.EXPECT().
				Continue(gomock.Any(), u, draft, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("a callback skips every later edit/city-draft continuation's repo lookup", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "menu:add_training"}

		err := run(t, u, in, func(m mocks) {
			m.drafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			m.scheduleDrafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			m.competitionDrafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			// No m.competitionEdits/m.catalogCities expectations here — a
			// callback never reaches those continuations' own repo
			// lookups.
			m.competitionMerges.EXPECT().
				GetMergeDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			// Reaching training's own addTrigger is what proves this link
			// skipped.
			m.create.EXPECT().
				Begin(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("no active competition draft (free text) skips to the next link", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "hi"}

		err := run(t, u, in, func(m mocks) {
			noOtherDrafts(m)

			m.competitionDrafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			m.competitionEdits.EXPECT().
				GetEditDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			m.competitionMerges.EXPECT().
				GetMergeDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			m.catalogCities.EXPECT().
				GetCityDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			// Reaching onboarding (the chain's last link, the catch-all
			// fallback) is what proves every continuation and trigger
			// skipped a plain, unrelated message.
			m.onboarding.EXPECT().
				Handle(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("repo error propagates and stops the chain", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "hi"}

		err := run(t, u, in, func(m mocks) {
			noOtherDrafts(m)

			m.competitionDrafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(nil, errors.New("fail"))
		})

		assert.Error(t, err)
	})
}
