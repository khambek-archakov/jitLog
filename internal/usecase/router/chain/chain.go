// Package chain builds Router's chain of responsibility — the ordered list
// of handlers that decides which scenario an update belongs to. Dependencies
// holds every scenario dependency as named fields rather than a positional
// argument list — a transposed pair of same-shaped dependencies (e.g.
// TrainingInfo/ScheduleInfo, both just Handle(ctx, userID, in) error) then
// fails loudly at the call site instead of compiling silently wrong, the
// way 17 positional constructor arguments invited. Chain.Handle is the only
// exported entry point Router needs — the list's own shape (which link
// comes first, which comes last) is an internal detail Router never has to
// know about. defaultOrder is the single place that knows the correct order
// (the onboarding gate first, active-draft continuations before every
// trigger, the raw onboarding dependency itself last as the catch-all
// fallback). training/schedule still live in their own sub-packages
// (router/chain/training, router/chain/schedule) — each is a real,
// growing feature domain with its own internal step ordering worth
// protecting behind a package boundary. onboarding/stats/profile/the
// continuations policy do not meet that bar (a single `if` each, or — for
// continuations — a router-level policy rather than a domain), so they
// live here as plain files instead of one-trigger packages.
package chain

import (
	"context"
	"errors"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/router/chain/schedule"
	"github.com/khambek-archakov/jitLog/internal/usecase/router/chain/training"
)

// Handler is one link in the chain. Returning model.ErrSkip means "not my
// case, try the next link"; any other return value (nil for success, or a
// real error) means this link handled the input and the walk should stop.
type Handler interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// Dependencies holds every scenario dependency chain.New needs.
type Dependencies struct {
	Onboarding        onboarding
	TrainingCreate    trainingCreate
	TrainingInfo      trainingInfo
	TrainingHistory   trainingHistory
	TrainingUpdate    trainingUpdate
	TrainingDelete    trainingDelete
	TrainingStats     trainingStats
	Profile           profile
	ScheduleCreate    scheduleCreate
	ScheduleList      scheduleList
	ScheduleInfo      scheduleInfo
	ScheduleUpdate    scheduleUpdate
	ScheduleDelete    scheduleDelete
	TrainingDraft     trainingDraft
	TrainingEditDraft trainingEditDraft
	ScheduleDraft     scheduleDraft
	ScheduleEditDraft scheduleEditDraft
	ProfileEditDraft  profileEditDraft
}

// Chain holds the scenario dependencies needed to assemble a chain of
// responsibility — see Handle.
type Chain struct {
	d Dependencies
}

func New(d Dependencies) *Chain {
	return &Chain{d: d}
}

// defaultOrder returns the chain in its one correct order. Only three
// relationships actually matter here — onboardingGate first, the raw
// onboarding dependency itself last (as the fallback — onboarding's own
// Handle already satisfies Handler, no wrapper needed), continuations
// before every trigger — everything else (the relative order of
// training/schedule/stats/profile) is free, since their callback prefixes
// never overlap. Unexported: nothing outside Handle needs this list's own
// shape, Router included.
func (c *Chain) defaultOrder() []Handler {
	d := c.d

	return []Handler{
		&onboardingGate{onboarding: d.Onboarding},
		continuations(d),
		training.New(d.TrainingCreate, d.TrainingInfo, d.TrainingHistory, d.TrainingUpdate, d.TrainingDelete),
		schedule.New(d.ScheduleCreate, d.ScheduleList, d.ScheduleInfo, d.ScheduleUpdate, d.ScheduleDelete),
		&statsTrigger{stats: d.TrainingStats},
		&profileTrigger{profile: d.Profile},
		d.Onboarding,
	}
}

// Handle walks the chain and returns the first non-model.ErrSkip outcome,
// or nil if nothing claims the input (never happens in practice — the raw
// onboarding dependency is always the last, never-skipping link — but kept
// for the same reason Router's old loop had it: an empty/fully-skipped
// chain is a no-op, not an error). This is the only exported entry point
// Chain needs; Router calls it directly and never needs to know the list's
// own shape.
func (c *Chain) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	for _, h := range c.defaultOrder() {
		if err := h.Handle(ctx, u, in); !errors.Is(err, model.ErrSkip) {
			return err
		}
	}

	return nil
}
