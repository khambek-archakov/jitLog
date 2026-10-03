package schedule_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/router/chain/schedule"
)

type mocks struct {
	create *MockscheduleCreate
	list   *MockscheduleList
	info   *MockscheduleInfo
	update *MockscheduleUpdate
	del    *MockscheduleDelete
}

// run builds a fresh mock set, lets prepare set expectations on it, then
// walks the real schedule.New(...) domain against u/in and returns the
// outcome — mirrors training's own domain_test.go.
func run(t *testing.T, u *model.User, in dto.Input, prepare func(m mocks)) error {
	t.Helper()

	ctrl := gomock.NewController(t)

	m := mocks{
		create: NewMockscheduleCreate(ctrl),
		list:   NewMockscheduleList(ctrl),
		info:   NewMockscheduleInfo(ctrl),
		update: NewMockscheduleUpdate(ctrl),
		del:    NewMockscheduleDelete(ctrl),
	}

	prepare(m)

	d := schedule.New(m.create, m.list, m.info, m.update, m.del)

	return d.Handle(context.Background(), u, in)
}

func TestDomain(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	t.Run("schedule:add begins a schedule draft", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:add"}

		err := run(t, u, in, func(m mocks) {
			m.create.EXPECT().
				Begin(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("schedule:view:{id} opens the card", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:view:7"}

		err := run(t, u, in, func(m mocks) {
			m.info.EXPECT().
				Handle(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("schedule:edit:* delegates to update.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:edit:7"}

		err := run(t, u, in, func(m mocks) {
			m.update.EXPECT().
				Handle(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("schedule:delete:* delegates to delete.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:delete:7"}

		err := run(t, u, in, func(m mocks) {
			m.del.EXPECT().
				Handle(gomock.Any(), userID, in).
				Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("schedule:list delegates to list.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "schedule:list"}

		err := run(t, u, in, func(m mocks) {
			m.list.EXPECT().
				Handle(gomock.Any(), userID, in).
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
