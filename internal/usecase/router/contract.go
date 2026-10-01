//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package router

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
)

// onboarding is what Router needs from the onboarding scenario — it knows
// nothing about training, and Router doesn't resolve it: onboarding just
// handles the input for an already-known user.
type onboarding interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// trainingCreate is what Router needs from the "add a training" dialog.
// Begin starts a fresh draft (Router only calls it on the specific menu
// trigger); Continue dispatches into an already-existing one.
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

type user interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error)
	Create(ctx context.Context, telegramID int64) (*model.User, error)
}

type trainingDraft interface {
	GetDraftByUserID(ctx context.Context, userID int64) (*model.TrainingDraft, error)
}

type trainingEditDraft interface {
	GetEditDraftByUserID(ctx context.Context, userID int64) (*model.TrainingEditDraft, error)
}
