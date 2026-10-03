//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package steps

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
	// AddBeltPromotion mirrors internal/usecase/profile's own private
	// contract — onboarding's own belt step uses it too, to backfill the
	// very first belt_promotion row once the user picks their starting belt.
	AddBeltPromotion(ctx context.Context, userID int64, belt model.Belt, promotedAt time.Time) error
}
