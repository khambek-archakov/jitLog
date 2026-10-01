package chain_test

import (
	"context"
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/router/chain"
)

// Shared test infrastructure for every link's own *_test.go file in this
// package — each link is tested by walking the real, fully assembled
// chain.New(...).Default() rather than constructing the (unexported) link
// type directly. gomock's strict expectations give the isolation this
// needs: a mock call that shouldn't happen (because the link under test
// was supposed to skip, or supposed to stop) fails the test immediately as
// an unexpected call.

type mocks struct {
	onboarding *Mockonboarding
	create     *MocktrainingCreate
	info       *MocktrainingInfo
	history    *MocktrainingHistory
	update     *MocktrainingUpdate
	delete     *MocktrainingDelete
	drafts     *MocktrainingDraft
	edits      *MocktrainingEditDraft
}

// noActiveDrafts stubs both the training draft and edit draft lookups to
// "none in progress" — the shared setup every case reaching the
// callback-prefix triggers needs.
func noActiveDrafts(m mocks, userID int64) {
	m.drafts.EXPECT().
		GetDraftByUserID(gomock.Any(), userID).
		Return(nil, model.ErrNotFound)

	m.edits.EXPECT().
		GetEditDraftByUserID(gomock.Any(), userID).
		Return(nil, model.ErrNotFound)
}

// dispatch mirrors Router.Route's own loop — Chain.Default's order is only
// meaningful when walked exactly like Route walks it.
func dispatch(ctx context.Context, handlers []chain.Handler, u *model.User, in dto.Input) error {
	for _, h := range handlers {
		if err := h.Handle(ctx, u, in); !errors.Is(err, chain.ErrSkip) {
			return err
		}
	}

	return nil
}

// run builds a fresh mock set, lets prepare set expectations on it, then
// walks the real Default() chain against u/in and returns the outcome.
func run(t *testing.T, u *model.User, in dto.Input, prepare func(m mocks)) error {
	t.Helper()

	ctrl := gomock.NewController(t)

	m := mocks{
		onboarding: NewMockonboarding(ctrl),
		create:     NewMocktrainingCreate(ctrl),
		info:       NewMocktrainingInfo(ctrl),
		history:    NewMocktrainingHistory(ctrl),
		update:     NewMocktrainingUpdate(ctrl),
		delete:     NewMocktrainingDelete(ctrl),
		drafts:     NewMocktrainingDraft(ctrl),
		edits:      NewMocktrainingEditDraft(ctrl),
	}

	prepare(m)

	handlers := chain.New(
		m.onboarding, m.create, m.info, m.history, m.update, m.delete, m.drafts, m.edits,
	).Default()

	return dispatch(context.Background(), handlers, u, in)
}
