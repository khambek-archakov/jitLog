package chain_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestDraftContinuation(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	t.Run("active draft delegates to create.Continue", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "60"}
		draft := &model.TrainingDraft{ID: 1, UserID: userID}

		err := run(t, u, in, func(m mocks) {
			m.drafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(draft, nil)

			m.create.EXPECT().
				Continue(gomock.Any(), draft, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("no active draft skips to the next link", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "45"}

		err := run(t, u, in, func(m mocks) {
			m.drafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(nil, model.ErrNotFound)

			// Reaching editDraftContinuation's own lookup is what proves
			// this link skipped rather than mishandling a "not found".
			m.edits.EXPECT().
				GetEditDraftByUserID(gomock.Any(), userID).
				Return(&model.TrainingEditDraft{UserID: userID, TrainingID: 7}, nil)

			m.update.EXPECT().
				Continue(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("repo error propagates and stops the chain", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasMessage: true, Text: "hi"}

		err := run(t, u, in, func(m mocks) {
			m.drafts.EXPECT().
				GetDraftByUserID(gomock.Any(), userID).
				Return(nil, errors.New("fail"))
		})

		assert.Error(t, err)
	})
}
