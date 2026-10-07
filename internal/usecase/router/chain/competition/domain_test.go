package competition_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/router/chain/competition"
)

type mocks struct {
	create   *MockcompetitionCreate
	list     *MockcompetitionList
	history  *MockcompetitionHistory
	info     *MockcompetitionInfo
	update   *MockcompetitionUpdate
	del      *MockcompetitionDelete
	submit   *MockcompetitionSubmit
	moderate *MockcompetitionModerate
}

// run builds a fresh mock set, lets prepare set expectations on it, then
// walks the real competition.New(...) domain against u/in and returns the
// outcome — mirrors schedule's own domain_test.go.
func run(t *testing.T, u *model.User, in dto.Input, prepare func(m mocks)) error {
	t.Helper()

	ctrl := gomock.NewController(t)

	m := mocks{
		create:   NewMockcompetitionCreate(ctrl),
		list:     NewMockcompetitionList(ctrl),
		history:  NewMockcompetitionHistory(ctrl),
		info:     NewMockcompetitionInfo(ctrl),
		update:   NewMockcompetitionUpdate(ctrl),
		del:      NewMockcompetitionDelete(ctrl),
		submit:   NewMockcompetitionSubmit(ctrl),
		moderate: NewMockcompetitionModerate(ctrl),
	}

	prepare(m)

	d := competition.New(m.create, m.list, m.history, m.info, m.update, m.del, m.submit, m.moderate)

	return d.Handle(context.Background(), u, in)
}

func TestDomain(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	t.Run("competition:add begins a tournament draft", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:add"}

		err := run(t, u, in, func(m mocks) {
			m.create.EXPECT().
				Begin(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("competition:view:{id} opens the card", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:view:7"}

		err := run(t, u, in, func(m mocks) {
			m.info.EXPECT().
				Handle(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("competition:edit:* delegates to update.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:edit:7"}

		err := run(t, u, in, func(m mocks) {
			m.update.EXPECT().
				Handle(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("competition:delete:* delegates to delete.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:delete:7"}

		err := run(t, u, in, func(m mocks) {
			m.del.EXPECT().
				Handle(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("competition:list delegates to list.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:list"}

		err := run(t, u, in, func(m mocks) {
			m.list.EXPECT().
				Handle(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("competition:history:page:{n} delegates to history.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:history:page:1"}

		err := run(t, u, in, func(m mocks) {
			m.history.EXPECT().
				Handle(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("competition:submit:* delegates to submit.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:submit:7"}

		err := run(t, u, in, func(m mocks) {
			m.submit.EXPECT().
				Handle(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("competition:moderate:* delegates to moderate.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "competition:moderate:7:approve"}

		err := run(t, u, in, func(m mocks) {
			m.moderate.EXPECT().
				Handle(gomock.Any(), u, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("callback matching nothing in this domain returns ErrSkip", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "training:view:7"}

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
