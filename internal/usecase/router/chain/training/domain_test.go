package training_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/router/chain/training"
)

type mocks struct {
	create  *MocktrainingCreate
	info    *MocktrainingInfo
	history *MocktrainingHistory
	update  *MocktrainingUpdate
	del     *MocktrainingDelete
}

// run builds a fresh mock set, lets prepare set expectations on it, then
// walks the real training.New(...) domain against u/in and returns the
// outcome — same "walk the real assembled thing" philosophy as the parent
// chain package's own tests, just scoped to this domain's own 5 triggers.
func run(t *testing.T, u *model.User, in dto.Input, prepare func(m mocks)) error {
	t.Helper()

	ctrl := gomock.NewController(t)

	m := mocks{
		create:  NewMocktrainingCreate(ctrl),
		info:    NewMocktrainingInfo(ctrl),
		history: NewMocktrainingHistory(ctrl),
		update:  NewMocktrainingUpdate(ctrl),
		del:     NewMocktrainingDelete(ctrl),
	}

	prepare(m)

	d := training.New(m.create, m.info, m.history, m.update, m.del)

	return d.Handle(context.Background(), u, in)
}

func TestDomain(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	t.Run("menu:add_training begins create", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "menu:add_training"}

		err := run(t, u, in, func(m mocks) {
			m.create.EXPECT().
				Begin(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("training:view:{id} opens the card", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:view:7"}

		err := run(t, u, in, func(m mocks) {
			m.info.EXPECT().
				Handle(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("training:history:page:{n} shows the list", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:history:page:0"}

		err := run(t, u, in, func(m mocks) {
			m.history.EXPECT().
				Handle(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("training:edit:* delegates to update.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:edit:7"}

		err := run(t, u, in, func(m mocks) {
			m.update.EXPECT().
				Handle(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("training:delete:* delegates to delete.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:delete:7"}

		err := run(t, u, in, func(m mocks) {
			m.del.EXPECT().
				Handle(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("callback matching nothing in this domain returns ErrSkip", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:add"}

		err := run(t, u, in, func(m mocks) {})

		assert.ErrorIs(t, err, model.ErrSkip)
	})

	t.Run("no callback at all returns ErrSkip", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID}

		err := run(t, u, in, func(m mocks) {})

		assert.ErrorIs(t, err, model.ErrSkip)
	})
}
