package update

import (
	"context"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

// UseCase is the onboarding scenario this handler translates and dispatches
// updates to.
type UseCase interface {
	Start(ctx context.Context, in dto.Input) error
}

// Handler consumes raw Telegram updates, translates each into a
// transport-neutral dto.Input and dispatches it to the usecase.
type Handler struct {
	useCase UseCase
	logger  *slog.Logger
}

func New(useCase UseCase, logger *slog.Logger) *Handler {
	return &Handler{useCase: useCase, logger: logger}
}

// Handle reads from updates until ctx is done.
func (h *Handler) Handle(ctx context.Context, updates tgbotapi.UpdatesChannel) {
	for {
		select {

		case <-ctx.Done():
			return

		case upd := <-updates:

			if upd.Message == nil && upd.CallbackQuery == nil {
				continue
			}

			if err := h.useCase.Start(ctx, NewInput(upd)); err != nil {
				h.logger.Error("failed to handle update", "error", err)
			}
		}
	}
}
