//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package chain

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// onboarding is what the chain needs from the onboarding scenario — it
// knows nothing about training, it just handles the input for an
// already-known user.
type onboarding interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// trainingCreate is what the chain needs from the "add a training" dialog.
// Begin starts a fresh draft; Continue dispatches into an existing one.
type trainingCreate interface {
	Begin(ctx context.Context, userID int64, in dto.Input) error
	Continue(ctx context.Context, d *model.TrainingDraft, in dto.Input) error
}

// trainingInfo shows a single training's card (training:view:{id}).
type trainingInfo interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// trainingHistory shows the paginated list of logged trainings
// (training:history:page:{n}).
type trainingHistory interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// trainingUpdate edits a single field of an existing training
// (training:edit:*). Handle covers every callback-driven step; Continue
// only fires for the two fields whose value arrives as free text
// (duration's "Другое" and notes), once a pending edit draft exists.
type trainingUpdate interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
	Continue(ctx context.Context, d *model.TrainingEditDraft, in dto.Input) error
}

// trainingDelete owns training:delete:* (a confirm screen, then the
// actual delete) — also fully stateless, same as most of trainingUpdate.
type trainingDelete interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// trainingStats shows the "📊 Статистика" screen (stats:period:*).
type trainingStats interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// profile shows the "👤 Профиль" screen and its belt-change flow
// (profile:*). Needs the full *model.User, same as onboarding — it reads
// and mutates it directly, not just the ID.
type profile interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// scheduleCreate is what the chain needs from the "add a schedule slot"
// dialog — same Begin/Continue shape as trainingCreate.
type scheduleCreate interface {
	Begin(ctx context.Context, userID int64, in dto.Input) error
	Continue(ctx context.Context, d *model.ScheduleDraft, in dto.Input) error
}

// scheduleList shows the "📅 Расписание" screen (schedule:list).
type scheduleList interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// scheduleInfo shows a single slot's card (schedule:view:{id}).
type scheduleInfo interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// scheduleUpdate edits a single field of an existing slot
// (schedule:edit:*). Handle covers every callback-driven step; Continue
// only fires for time's "Другое" fallback, once a pending edit draft
// exists.
type scheduleUpdate interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
	Continue(ctx context.Context, d *model.ScheduleEditDraft, in dto.Input) error
}

// scheduleDelete owns schedule:delete:* (a confirm screen, then the
// actual delete) — also fully stateless, same as most of scheduleUpdate.
type scheduleDelete interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

type trainingDraft interface {
	GetDraftByUserID(ctx context.Context, userID int64) (*model.TrainingDraft, error)
}

type trainingEditDraft interface {
	GetEditDraftByUserID(ctx context.Context, userID int64) (*model.TrainingEditDraft, error)
}

type scheduleDraft interface {
	GetDraftByUserID(ctx context.Context, userID int64) (*model.ScheduleDraft, error)
}

type scheduleEditDraft interface {
	GetEditDraftByUserID(ctx context.Context, userID int64) (*model.ScheduleEditDraft, error)
}
