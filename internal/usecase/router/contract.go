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

// training is what Router needs from the "add a training" scenario. Begin
// starts a fresh draft (Router only calls it on the specific menu trigger);
// Continue dispatches into an already-existing one.
type training interface {
	Begin(ctx context.Context, userID int64, in dto.Input) error
	Continue(ctx context.Context, d *model.TrainingDraft, in dto.Input) error
}

type user interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error)
	Create(ctx context.Context, telegramID int64) (*model.User, error)
}

type trainingDraft interface {
	GetDraftByUserID(ctx context.Context, userID int64) (*model.TrainingDraft, error)
}
