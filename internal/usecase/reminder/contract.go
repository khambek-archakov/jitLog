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
	SendWithKeyboardReturningID(ctx context.Context, chatID int64, text string, keyboard dto.Keyboard) (int, error)
	EditMessageWithKeyboard(ctx context.Context, chatID int64, messageID int, text string, keyboard dto.Keyboard) error
	RemoveKeyboard(ctx context.Context, chatID int64, messageID int) error
	AnswerCallback(ctx context.Context, callbackID string) error
	AnswerCallbackWithText(ctx context.Context, callbackID, text string) error
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
	// GetSlot is only needed by the rec: callback — the job's own two
	// walks (weekly, post-session) already have every slot from ListSlots.
	GetSlot(ctx context.Context, id int64) (*model.ScheduleSlot, error)
}

type trainingRepo interface {
	HasTrainingBetween(ctx context.Context, userID int64, from, to time.Time) (bool, error)
	// CountTrainingsOnDate backs both the post-session job's own "fewer
	// trainings logged than scheduled slots so far today" check and the
	// rec: callback's identical repeat-tap guard.
	CountTrainingsOnDate(ctx context.Context, userID int64, date time.Time) (int, error)
	// LatestTrainingOnDate is the rec: callback's own "already recorded,
	// show that one instead of creating a duplicate" fallback.
	LatestTrainingOnDate(ctx context.Context, userID int64, date time.Time) (*model.Training, error)
	CreateTraining(
		ctx context.Context, userID int64, date time.Time, trainingType model.TrainingType, durationMinutes int32,
		notes *string,
	) (*model.Training, error)
}

// sentRepo mirrors the reminder_log table — this job's own idempotency
// marker, checked/written instead of any in-memory state so a process
// restart mid-window still can't send twice.
type sentRepo interface {
	HasSent(ctx context.Context, userID int64, kind model.ReminderKind, refDate time.Time, slotID int64) (bool, error)
	MarkSent(ctx context.Context, userID int64, kind model.ReminderKind, refDate time.Time, slotID, messageID int64) error
	// CountSentBetween backs the "at most two reminders per local day,
	// weekly and post-session combined" cap.
	CountSentBetween(ctx context.Context, userID int64, from, to time.Time) (int, error)
	// LastMessageID is the post-session job's own "strip the previous
	// reminder's keyboard before sending a new one" lookup.
	LastMessageID(ctx context.Context, userID int64) (int64, bool, error)
}
