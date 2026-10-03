package steps

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/menu"
)

const welcomeBack = "С возвращением!\n\n" + menu.Text

// callbackMenuBack mirrors internal/usecase/training/history's own private
// constant — it's how a history/card/stats/schedule screen gets back to
// the main menu.
const callbackMenuBack = "menu:back"

// CompletedStep only re-shows the main menu, so it needs no user repository access.
type CompletedStep struct {
	bot sender
}

func NewCompleted(bot sender) *CompletedStep {
	return &CompletedStep{bot: bot}
}

func (s *CompletedStep) Handle(ctx context.Context, _ *model.User, in dto.Input) error {
	if in.IsStartCmd {
		return sendMainMenu(ctx, s.bot, in.ChatID, welcomeBack)
	}

	if in.HasCallback {
		switch in.CallbackData {
		// callbackMenuAddTraining, callbackMenuMyTrainings, stats:period:*,
		// profile:*, schedule:*, training:view:*, training:edit:* and
		// training:delete:* are intercepted upstream by the router before
		// they ever reach here (see internal/usecase/router).
		case callbackMenuBack:
			if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
				return err
			}

			return sendMainMenu(ctx, s.bot, in.ChatID, menu.Text)
		default:
			return s.bot.AnswerCallback(ctx, in.CallbackID)
		}
	}

	if in.HasMessage {
		return sendMainMenu(ctx, s.bot, in.ChatID, menu.Text)
	}

	return nil
}

func sendMainMenu(ctx context.Context, bot sender, chatID int64, text string) error {
	return bot.SendWithKeyboard(ctx, chatID, text, menu.Keyboard())
}
