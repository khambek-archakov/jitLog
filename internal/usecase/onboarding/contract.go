//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package onboarding

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

type user interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error)
	Create(ctx context.Context, telegramID int64) (*model.User, error)
	Update(ctx context.Context, u *model.User) error
	AddBeltPromotion(ctx context.Context, userID int64, belt model.Belt, promotedAt time.Time) error
}

// handler mirrors steps' own (also private) step-handler contract — UseCase
// is the only thing that needs to name this type, to keep its dispatch map.
type handler interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}
