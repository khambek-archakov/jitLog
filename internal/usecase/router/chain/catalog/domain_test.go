package catalog_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/router/chain/catalog"
)

type mocks struct {
	list *MockcatalogList
	info *MockcatalogInfo
	add  *MockcatalogAdd
}

func run(t *testing.T, u *model.User, in dto.Input, prepare func(m mocks)) error {
	t.Helper()

	ctrl := gomock.NewController(t)

	m := mocks{
		list: NewMockcatalogList(ctrl),
		info: NewMockcatalogInfo(ctrl),
		add:  NewMockcatalogAdd(ctrl),
	}

	prepare(m)

	d := catalog.New(m.list, m.info, m.add)

	return d.Handle(context.Background(), u, in)
}

func TestDomain(t *testing.T) {
	t.Parallel()

	const userID, chatID int64 = 42, 777

	u := &model.User{ID: userID, OnboardingStep: model.OnboardingStepCompleted}

	t.Run("catalog:list delegates to list.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:list"}

		err := run(t, u, in, func(m mocks) {
			m.list.EXPECT().Handle(gomock.Any(), u, in).Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("catalog:list:page:{n} delegates to list.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:list:page:1"}

		err := run(t, u, in, func(m mocks) {
			m.list.EXPECT().Handle(gomock.Any(), u, in).Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("catalog:city:prompt delegates to list.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:city:prompt"}

		err := run(t, u, in, func(m mocks) {
			m.list.EXPECT().Handle(gomock.Any(), u, in).Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("catalog:city:all delegates to list.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:city:all"}

		err := run(t, u, in, func(m mocks) {
			m.list.EXPECT().Handle(gomock.Any(), u, in).Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("catalog:view:{id} opens the card", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:view:7"}

		err := run(t, u, in, func(m mocks) {
			m.info.EXPECT().Handle(gomock.Any(), u, in).Return(nil)
		})

		assert.NoError(t, err)
	})

	t.Run("catalog:add:{id} delegates to add.Handle", func(t *testing.T) {
		t.Parallel()

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "catalog:add:7"}

		err := run(t, u, in, func(m mocks) {
			m.add.EXPECT().Handle(gomock.Any(), u, in).Return(nil)
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
