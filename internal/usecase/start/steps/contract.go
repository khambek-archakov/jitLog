package steps

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

type Sender interface {
	Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
	Request(c tgbotapi.Chattable) (*tgbotapi.APIResponse, error)
}

type User interface {
	GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error)
	Create(ctx context.Context, telegramID int64) (*model.User, error)
	Update(ctx context.Context, u *model.User) error
}

// Handler reacts to one onboarding step. It knows nothing about tgbotapi —
// it only sees the transport-neutral dto.Input built by the delivery layer.
type Handler interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}
