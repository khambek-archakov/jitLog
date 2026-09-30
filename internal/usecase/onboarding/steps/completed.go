package steps

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
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

// callbackTrainingEditPrefix and callbackTrainingDeletePrefix mirror
// internal/usecase/training/steps' own private constants — taps on a
// finished training's "Изменить"/"Удалить" buttons land here (no draft is
// active anymore by then, so the router falls back to onboarding).
const (
	callbackTrainingEditPrefix   = "training:edit:"
	callbackTrainingDeletePrefix = "training:delete:"
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
		switch {
		// callbackMenuAddTraining is intercepted upstream by the router
		// before it ever reaches here (see internal/usecase/router) — a
		// training_draft exists by the time this step could see it again.
		case in.CallbackData == callbackMenuSchedule, in.CallbackData == callbackMenuStats:
			return s.bot.AnswerCallbackWithText(in.CallbackID, comingSoonText)
		case strings.HasPrefix(in.CallbackData, callbackTrainingEditPrefix),
			strings.HasPrefix(in.CallbackData, callbackTrainingDeletePrefix):
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
		dto.Row(dto.Button{Label: "🥋 Добавить тренировку", Data: callbackMenuAddTraining}),
		dto.Row(dto.Button{Label: "📅 Расписание", Data: callbackMenuSchedule}),
		dto.Row(dto.Button{Label: "📊 Статистика", Data: callbackMenuStats}),
	}
}
