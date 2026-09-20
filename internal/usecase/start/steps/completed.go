package steps

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

const menuPromptText = "Вот что я умею:"

const (
	welcomeBack    = "С возвращением!\n\n" + menuPromptText
	comingSoonText = "Скоро!"
)

const (
	callbackMenuAddTraining = "menu:add_training"
	callbackMenuSchedule    = "menu:schedule"
	callbackMenuStats       = "menu:stats"
)

// CompletedStep only re-shows the main menu, so it needs no user repository access.
type CompletedStep struct {
	bot sender
}

func NewCompleted(bot sender) *CompletedStep {
	return &CompletedStep{bot: bot}
}

func (s *CompletedStep) Handle(_ context.Context, _ *model.User, in dto.Input) error {
	if in.IsStartCmd {
		return sendMainMenu(s.bot, in.ChatID, welcomeBack)
	}

	if in.HasCallback {
		switch in.CallbackData {
		case callbackMenuAddTraining, callbackMenuSchedule, callbackMenuStats:
			return s.bot.AnswerCallbackWithText(in.CallbackID, comingSoonText)
		default:
			return s.bot.AnswerCallback(in.CallbackID)
		}
	}

	if in.HasMessage {
		return sendMainMenu(s.bot, in.ChatID, menuPromptText)
	}

	return nil
}

func sendMainMenu(bot sender, chatID int64, text string) error {
	return bot.SendWithKeyboard(chatID, text, mainMenuKeyboard())
}

func mainMenuKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "Добавить тренировку", Data: callbackMenuAddTraining}),
		dto.Row(dto.Button{Label: "Расписание", Data: callbackMenuSchedule}),
		dto.Row(dto.Button{Label: "Статистика", Data: callbackMenuStats}),
	}
}
