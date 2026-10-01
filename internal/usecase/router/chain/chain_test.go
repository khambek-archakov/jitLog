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
	onboarding     *Mockonboarding
	create         *MocktrainingCreate
	info           *MocktrainingInfo
	history        *MocktrainingHistory
	update         *MocktrainingUpdate
	delete         *MocktrainingDelete
	stats          *MocktrainingStats
	profile        *Mockprofile
	scheduleCreate *MockscheduleCreate
	scheduleList   *MockscheduleList
	scheduleInfo   *MockscheduleInfo
	scheduleUpdate *MockscheduleUpdate
	scheduleDelete *MockscheduleDelete
	drafts         *MocktrainingDraft
	edits          *MocktrainingEditDraft
	scheduleDrafts *MockscheduleDraft
	scheduleEdits  *MockscheduleEditDraft
}

// noActiveDrafts stubs the training draft, training edit draft, schedule
// draft and schedule edit draft lookups to "none in progress" — the shared
// setup every case reaching the callback-prefix triggers needs.
func noActiveDrafts(m mocks, userID int64) {
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
		onboarding:     NewMockonboarding(ctrl),
		create:         NewMocktrainingCreate(ctrl),
		info:           NewMocktrainingInfo(ctrl),
		history:        NewMocktrainingHistory(ctrl),
		update:         NewMocktrainingUpdate(ctrl),
		delete:         NewMocktrainingDelete(ctrl),
		stats:          NewMocktrainingStats(ctrl),
		profile:        NewMockprofile(ctrl),
		scheduleCreate: NewMockscheduleCreate(ctrl),
		scheduleList:   NewMockscheduleList(ctrl),
		scheduleInfo:   NewMockscheduleInfo(ctrl),
		scheduleUpdate: NewMockscheduleUpdate(ctrl),
		scheduleDelete: NewMockscheduleDelete(ctrl),
		drafts:         NewMocktrainingDraft(ctrl),
		edits:          NewMocktrainingEditDraft(ctrl),
		scheduleDrafts: NewMockscheduleDraft(ctrl),
		scheduleEdits:  NewMockscheduleEditDraft(ctrl),
	}

	prepare(m)

	handlers := chain.New(
		m.onboarding, m.create, m.info, m.history, m.update, m.delete, m.stats, m.profile,
		m.scheduleCreate, m.scheduleList, m.scheduleInfo, m.scheduleUpdate, m.scheduleDelete,
		m.drafts, m.edits, m.scheduleDrafts, m.scheduleEdits,
	).Default()

	return dispatch(context.Background(), handlers, u, in)
}
