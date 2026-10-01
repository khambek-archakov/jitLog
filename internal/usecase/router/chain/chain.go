// Package chain builds Router's chain of responsibility — the ordered list
// of handlers that decides which scenario an update belongs to. Chain
// holds the scenario dependencies; its Default method is the single place
// that knows the correct order (state continuations before callback-prefix
// triggers, the onboarding fallback last), so main.go never has to make
// that call itself.
package chain

import (
	"context"
	"errors"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// ErrSkip is what a Handler returns when the input isn't its case —
// callers must check for it with errors.Is before treating a non-nil
// return as a real failure.
var ErrSkip = errors.New("skip")

// Handler is one link in the chain. Returning ErrSkip means "not my
// case, try the next link"; any other return value (nil for success, or a
// real error) means this link handled the input and the walk should stop.
type Handler interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// Chain holds the scenario dependencies needed to assemble a chain of
// responsibility — see Default.
type Chain struct {
	onboarding     onboarding
	create         trainingCreate
	info           trainingInfo
	history        trainingHistory
	update         trainingUpdate
	delete         trainingDelete
	stats          trainingStats
	profile        profile
	scheduleCreate scheduleCreate
	scheduleList   scheduleList
	scheduleInfo   scheduleInfo
	scheduleUpdate scheduleUpdate
	scheduleDelete scheduleDelete
	drafts         trainingDraft
	edits          trainingEditDraft
	scheduleDrafts scheduleDraft
	scheduleEdits  scheduleEditDraft
}

func New(
	onboarding onboarding,
	create trainingCreate,
	info trainingInfo,
	history trainingHistory,
	update trainingUpdate,
	del trainingDelete,
	stats trainingStats,
	prof profile,
	scheduleCreate scheduleCreate,
	scheduleList scheduleList,
	scheduleInfo scheduleInfo,
	scheduleUpdate scheduleUpdate,
	scheduleDelete scheduleDelete,
	drafts trainingDraft,
	edits trainingEditDraft,
	scheduleDrafts scheduleDraft,
	scheduleEdits scheduleEditDraft,
) *Chain {
	return &Chain{
		onboarding: onboarding, create: create, info: info, history: history, update: update, delete: del,
		stats: stats, profile: prof, scheduleCreate: scheduleCreate, scheduleList: scheduleList,
		scheduleInfo: scheduleInfo, scheduleUpdate: scheduleUpdate, scheduleDelete: scheduleDelete,
		drafts: drafts, edits: edits, scheduleDrafts: scheduleDrafts, scheduleEdits: scheduleEdits,
	}
}

// Default returns the app's chain of responsibility in its one correct
// order.
func (c *Chain) Default() []Handler {
	return []Handler{
		&onboardingGate{onboarding: c.onboarding},
		&draftContinuation{create: c.create, drafts: c.drafts},
		&editDraftContinuation{update: c.update, edits: c.edits},
		&scheduleDraftContinuation{create: c.scheduleCreate, drafts: c.scheduleDrafts},
		&scheduleEditDraftContinuation{update: c.scheduleUpdate, edits: c.scheduleEdits},
		&addTrainingTrigger{create: c.create},
		&viewTrigger{info: c.info},
		&historyTrigger{history: c.history},
		&editTrigger{update: c.update},
		&deleteTrigger{delete: c.delete},
		&statsTrigger{stats: c.stats},
		&profileTrigger{profile: c.profile},
		&scheduleAddTrigger{create: c.scheduleCreate},
		&scheduleViewTrigger{info: c.scheduleInfo},
		&scheduleEditTrigger{update: c.scheduleUpdate},
		&scheduleDeleteTrigger{delete: c.scheduleDelete},
		&scheduleListTrigger{list: c.scheduleList},
		&onboardingFallback{onboarding: c.onboarding},
	}
}
