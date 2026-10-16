//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package router

import (
	"context"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

type user interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error)
	Create(ctx context.Context, telegramID int64) (*model.User, error)
	// TouchActivity records the user's own activity middleware write — see
	// Router.touchActivity, the only caller.
	TouchActivity(ctx context.Context, userID int64, now time.Time) error
}

// chain is what Router needs from the assembled chain of responsibility —
// a single entry point. Declared privately here (rather than importing
// chain.Chain itself) per this codebase's small-interface-per-consumer
// convention, and because Router genuinely only needs the one method.
type chain interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}
