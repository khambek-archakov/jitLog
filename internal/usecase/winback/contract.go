//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package winback

import (
	"context"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

type sender interface {
	SendWithKeyboard(ctx context.Context, chatID int64, text string, keyboard dto.Keyboard) error
	SendAnimationWithKeyboard(ctx context.Context, chatID int64, fileID, caption string, keyboard dto.Keyboard) error
	RemoveKeyboard(ctx context.Context, chatID int64, messageID int) error
	AnswerCallback(ctx context.Context, callbackID string) error
}

// userRepo is what the winback job and its own button handler need from
// the user repository. ListAll drives each tick's walk — unlike
// reminder's ListRemindersEnabled, almost any user is a winback candidate,
// so the actual gating lives in Go (see Run/processUser), not a SQL filter.
type userRepo interface {
	ListAll(ctx context.Context) ([]*model.User, error)
	SetWinbackSent(ctx context.Context, userID int64, at time.Time) error
	SetWinbackDeclined(ctx context.Context, userID int64, at time.Time) error
}

// reminderRepo is just enough to check whether the weekly training
// reminder already reached this user recently — a winback nudge right on
// top of that would be noisy.
type reminderRepo interface {
	HasSentWithin(ctx context.Context, userID int64, since time.Time) (bool, error)
}
