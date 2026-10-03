//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package update

import (
	"context"
	"time"

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

type trainingRepo interface {
	GetTraining(ctx context.Context, id int64) (*model.Training, error)
	UpdateTraining(
		ctx context.Context,
		id int64,
		date time.Time,
		trainingType model.TrainingType,
		durationMinutes int32,
		notes *string,
	) (*model.Training, error)
	SetEditDraft(ctx context.Context, userID, trainingID int64, field model.TrainingEditField) error
	DeleteEditDraft(ctx context.Context, userID int64) error
}
