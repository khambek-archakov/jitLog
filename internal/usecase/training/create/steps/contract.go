//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package steps

import (
	"context"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// sender mirrors internal/usecase/onboarding/steps' own sender — same shape,
// duplicated deliberately rather than shared across scenario packages.
type sender interface {
	Send(ctx context.Context, chatID int64, text string) error
	SendWithKeyboard(ctx context.Context, chatID int64, text string, keyboard dto.Keyboard) error
	EditMessageWithKeyboard(ctx context.Context, chatID int64, messageID int, text string, keyboard dto.Keyboard) error
	AnswerCallback(ctx context.Context, callbackID string) error
	AnswerCallbackWithText(ctx context.Context, callbackID, text string) error
}

// draftRepo is what a step needs from the training_draft/training storage.
// Broader than any single step uses, mirroring the same convention as
// start/steps' user interface.
type draftRepo interface {
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
