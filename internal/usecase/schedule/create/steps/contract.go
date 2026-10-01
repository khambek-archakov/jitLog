//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package steps

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

type sender interface {
	Send(chatID int64, text string) error
	SendWithKeyboard(chatID int64, text string, keyboard dto.Keyboard) error
	EditMessageWithKeyboard(chatID int64, messageID int, text string, keyboard dto.Keyboard) error
	AnswerCallback(callbackID string) error
	AnswerCallbackWithText(callbackID, text string) error
}

// draftRepo is what a step needs from the schedule_draft/schedule_slot
// storage. Broader than any single step uses, mirroring the same
// convention as training/create/steps' own draftRepo.
type draftRepo interface {
	UpdateDraft(ctx context.Context, d *model.ScheduleDraft) error
	CreateSlot(
		ctx context.Context, userID int64, dayOfWeek, timeMinutes int16, trainingType model.TrainingType,
	) (*model.ScheduleSlot, error)
	DeleteDraft(ctx context.Context, userID int64) error
}
