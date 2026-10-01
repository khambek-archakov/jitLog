// Package menu is the single canonical definition of the main-menu
// message — text plus its inline keyboard. Any scenario that needs to
// re-show "what can I do" (onboarding's CompletedStep, and create's "saved"
// confirmation, which nudges the user toward it next) imports this instead
// of building its own copy, so the four buttons never drift between places.
package menu

import "github.com/khambek-archakov/jitLog/internal/usecase/dto"

const Text = "Вот что я умею:"

const (
	callbackAddTraining = "menu:add_training"
	callbackMyTrainings = "training:history:page:0"
	callbackSchedule    = "menu:schedule"
	callbackStats       = "menu:stats"
)

func Keyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "➕ Добавить тренировку", Data: callbackAddTraining}),
		dto.Row(dto.Button{Label: "🗒️ Мои тренировки", Data: callbackMyTrainings}),
		dto.Row(dto.Button{Label: "📅 Расписание", Data: callbackSchedule}),
		dto.Row(dto.Button{Label: "📊 Статистика", Data: callbackStats}),
	}
}
