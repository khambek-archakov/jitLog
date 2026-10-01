//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package update

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

type slotRepo interface {
	GetSlot(ctx context.Context, id int64) (*model.ScheduleSlot, error)
	UpdateSlot(
		ctx context.Context, id int64, dayOfWeek, timeMinutes int16, trainingType model.TrainingType,
	) (*model.ScheduleSlot, error)
	SetEditDraft(ctx context.Context, userID, slotID int64) error
	DeleteEditDraft(ctx context.Context, userID int64) error
}
