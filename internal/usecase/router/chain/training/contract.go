//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package training

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// handler is one trigger in this domain's own internal sequence — same
// shape as the parent chain package's own Handler, declared separately so
// this package never has to import chain (which would create a cycle,
// since chain itself imports this package to build its own handler list).
type handler interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// trainingCreate is what this domain needs from the "add a training"
// dialog — just enough to kick a fresh draft off. The parent chain
// package's own copy of this interface (used by its draftContinuation)
// also needs Continue, which this package never calls.
type trainingCreate interface {
	Begin(ctx context.Context, userID int64, in dto.Input) error
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
// (training:edit:*). This package only ever drives it by callback, never
// Continue — that's the parent chain package's own editDraftContinuation.
type trainingUpdate interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// trainingDelete owns training:delete:* (a confirm screen, then the
// actual delete).
type trainingDelete interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}
