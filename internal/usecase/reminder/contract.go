//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package reminder

import (
	"context"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

type sender interface {
	SendWithKeyboard(ctx context.Context, chatID int64, text string, keyboard dto.Keyboard) error
	EditMessageWithKeyboard(ctx context.Context, chatID int64, messageID int, text string, keyboard dto.Keyboard) error
	AnswerCallback(ctx context.Context, callbackID string) error
}

// userRepo is what the reminder job needs from the user repository —
// ListRemindersEnabled drives each tick's own walk, Update disables a
// blocked or opted-out user.
type userRepo interface {
	ListRemindersEnabled(ctx context.Context) ([]*model.User, error)
	Update(ctx context.Context, u *model.User) error
}

type slotRepo interface {
	ListSlots(ctx context.Context, userID int64) ([]*model.ScheduleSlot, error)
}

type trainingRepo interface {
	HasTrainingBetween(ctx context.Context, userID int64, from, to time.Time) (bool, error)
}

// sentRepo mirrors the reminder_sent table — this job's own idempotency
// marker, checked/written instead of any in-memory state so a process
// restart mid-window still can't send twice.
type sentRepo interface {
	HasSent(ctx context.Context, userID int64, weekStart time.Time) (bool, error)
	MarkSent(ctx context.Context, userID int64, weekStart time.Time) error
}
