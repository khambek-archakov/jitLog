package steps

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

const welcomeBack = "С возвращением!"

// CompletedStep only re-shows the main menu, so it needs no user repository access.
type CompletedStep struct {
	bot Sender
}

func NewCompleted(bot Sender) *CompletedStep {
	return &CompletedStep{bot: bot}
}

func (s *CompletedStep) Handle(_ context.Context, _ *model.User, in dto.Input) error {
	if !in.IsStartCmd {
		return nil
	}

	return sendMainMenu(s.bot, in.ChatID, welcomeBack)
}
