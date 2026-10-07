package chain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestCatalogCityDraftContinuation(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	// Every case here is already past every other continuation — this is
	// the last link before the callback-prefix triggers.
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

		m.competitionEdits.EXPECT().
			GetEditDraftByUserID(gomock.Any(), userID).
			Return(nil, model.ErrNotFound)

		m.competitionMerges.EXPECT().
			GetMergeDraftByUserID(gomock.Any(), userID).
			Return(nil, model.ErrNotFound)
	}

	t.Run("active city draft delegates to list.Continue", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "Казань"}
		draft := &model.CatalogCityDraft{UserID: userID}

		err := run(t, u, in, func(m mocks) {
			noOtherDrafts(m)

			m.catalogCities.EXPECT().
				GetCityDraftByUserID(gomock.Any(), userID).
				Return(draft, nil)

			m.catalogList.EXPECT().
				Continue(gomock.Any(), u, draft, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("a callback skips immediately without touching the repo at all", func(t *testing.T) {
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

			m.competitionMerges.EXPECT().
				GetMergeDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			// No m.catalogCities expectation here — a callback (this
			// continuation's own "❌ Отмена" button included) never
			// reaches its repo lookup at all; catalog's own listTrigger
			// dispatches it to Handle instead.
			// Reaching training's own addTrigger is what proves this link
			// skipped.
			m.create.EXPECT().
				Begin(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("no active city draft (free text) skips to the next link", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "hi"}

		err := run(t, u, in, func(m mocks) {
			noOtherDrafts(m)

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

			m.catalogCities.EXPECT().
				GetCityDraftByUserID(gomock.Any(), userID).
				Return(nil, errors.New("fail"))
		})

		assert.Error(t, err)
	})
}
