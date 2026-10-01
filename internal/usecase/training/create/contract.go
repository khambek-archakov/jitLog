//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package create

import (
	"context"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
)

type sender interface {
	Send(chatID int64, text string) error
	SendWithKeyboard(chatID int64, text string, keyboard dto.Keyboard) error
	EditMessageWithKeyboard(chatID int64, messageID int, text string, keyboard dto.Keyboard) error
	AnswerCallback(callbackID string) error
	AnswerCallbackWithText(callbackID, text string) error
}

type draftRepo interface {
	CreateDraft(ctx context.Context, userID int64) (*model.TrainingDraft, error)
	UpdateDraft(ctx context.Context, d *model.TrainingDraft) error
	CreateTraining(
		ctx context.Context,
		userID int64,
		date time.Time,
		trainingType model.TrainingType,
		durationMinutes int32,
		notes *string,
	) (*model.Training, error)
	DeleteDraft(ctx context.Context, userID int64) error
}

// handler mirrors steps' own (also private) step-handler contract — UseCase
// is the only thing that needs to name this type, to keep its dispatch map.
type handler interface {
	Handle(ctx context.Context, d *model.TrainingDraft, in dto.Input) error
}
