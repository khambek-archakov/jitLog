//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package delete

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

type slotRepo interface {
	GetSlot(ctx context.Context, id int64) (*model.ScheduleSlot, error)
	DeleteSlot(ctx context.Context, id int64) error
}
