//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package history

import (
	"context"

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

type trainingRepo interface {
	// ListTrainings returns up to limit trainings for userID starting at
	// offset, newest first, plus whether more exist past this page.
	ListTrainings(ctx context.Context, userID int64, limit, offset int) ([]*model.Training, bool, error)
}
