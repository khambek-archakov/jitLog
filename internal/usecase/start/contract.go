//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package start

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

type sender interface {
	Send(chatID int64, text string) error
	SendWithKeyboard(chatID int64, text string, keyboard dto.Keyboard) error
	AnswerCallback(callbackID string) error
	AnswerCallbackWithText(callbackID, text string) error
}

type user interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error)
	Create(ctx context.Context, telegramID int64) (*model.User, error)
	Update(ctx context.Context, u *model.User) error
}

// handler mirrors steps' own (also private) step-handler contract — UseCase
// is the only thing that needs to name this type, to keep its dispatch map.
type handler interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}
