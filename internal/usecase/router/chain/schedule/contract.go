//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package schedule

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// handler is one trigger in this domain's own internal sequence — see
// training's identical contract.go comment for why it's declared here
// rather than imported from the parent chain package.
type handler interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// scheduleCreate is what this domain needs from the "add a schedule slot"
// dialog — just enough to kick a fresh draft off.
type scheduleCreate interface {
	Begin(ctx context.Context, userID int64, in dto.Input) error
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
// (schedule:edit:*). This package only ever drives it by callback, never
// Continue — that's the parent chain package's own
// scheduleEditDraftContinuation.
type scheduleUpdate interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// scheduleDelete owns schedule:delete:* (a confirm screen, then the
// actual delete).
type scheduleDelete interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}
