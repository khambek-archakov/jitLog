//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package create

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

type sender interface {
	Send(ctx context.Context, chatID int64, text string) error
	SendWithKeyboard(ctx context.Context, chatID int64, text string, keyboard dto.Keyboard) error
	EditMessageWithKeyboard(ctx context.Context, chatID int64, messageID int, text string, keyboard dto.Keyboard) error
	AnswerCallback(ctx context.Context, callbackID string) error
	AnswerCallbackWithText(ctx context.Context, callbackID, text string) error
}

// draftRepo is the union of what UseCase needs directly (CreateDraft,
// DeleteDraft) and what it hands down to steps.NewDay/NewTime/NewType
// (UpdateDraft, CreateSlot) — the same repo value flows through both, so
// its interface here has to satisfy steps' own (separately declared)
// draftRepo too.
type draftRepo interface {
	CreateDraft(ctx context.Context, userID int64) (*model.ScheduleDraft, error)
	UpdateDraft(ctx context.Context, d *model.ScheduleDraft) error
	CreateSlot(
		ctx context.Context, userID int64, dayOfWeek, timeMinutes int16, trainingType model.TrainingType,
	) (*model.ScheduleSlot, error)
	DeleteDraft(ctx context.Context, userID int64) error
}

// handler mirrors steps' own (also private) step-handler contract — UseCase
// is the only thing that needs to name this type, to keep its dispatch map.
type handler interface {
	Handle(ctx context.Context, d *model.ScheduleDraft, in dto.Input) error
}
