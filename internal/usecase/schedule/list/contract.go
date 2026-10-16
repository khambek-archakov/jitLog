//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package list

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
	ListSlots(ctx context.Context, userID int64) ([]*model.ScheduleSlot, error)
}

// userRepo is just enough to persist the reminder toggle and timezone pick
// — the full *model.User to read/mutate comes from the chain already (see
// router/chain/schedule's listTrigger).
type userRepo interface {
	Update(ctx context.Context, u *model.User) error
}
