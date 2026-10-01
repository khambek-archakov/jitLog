//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package stats

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

type trainingRepo interface {
	// ListAllTrainings returns userID's full training history, oldest
	// first — stats needs it all, both for the period aggregates and for
	// the streak, which spans further back than any single period.
	ListAllTrainings(ctx context.Context, userID int64) ([]*model.Training, error)
}
