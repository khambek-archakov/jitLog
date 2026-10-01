package info

import (
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	callbackEditPrefix   = "schedule:edit:"
	callbackDeletePrefix = "schedule:delete:"
	// callbackScheduleList mirrors internal/usecase/schedule/list's own
	// private constant — the card's "Назад" button goes back there.
	callbackScheduleList = "schedule:list"
)

var weekdayLabels = [...]string{"Пн", "Вт", "Ср", "Чт", "Пт", "Сб", "Вс"}

// Card is the full single-slot view: a headline plus Body.
func Card(s *model.ScheduleSlot) string {
	return "Слот расписания\n\n" + Body(s)
}

// Body formats a slot's day/time/type — shared by the card view and
// create's confirmation screen.
func Body(s *model.ScheduleSlot) string {
	return fmt.Sprintf("%s — %s, %s", DayLabel(s.DayOfWeek), FormatTime(s.TimeMinutes), TrainingTypeLabel(s.TrainingType))
}

// Keyboard is the card's own layout: Изменить and Удалить each on their
// own row, plus a row back to the schedule list.
func Keyboard(slotID int64) dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "✏️ Изменить", Data: fmt.Sprintf("%s%d", callbackEditPrefix, slotID)}),
		dto.Row(dto.Button{Label: "🗑️ Удалить", Data: fmt.Sprintf("%s%d", callbackDeletePrefix, slotID)}),
		dto.Row(dto.Button{Label: "← Назад", Data: callbackScheduleList}),
	}
}

func DayLabel(day int16) string {
	if day < 1 || day > 7 {
		return "—"
	}

	return weekdayLabels[day-1]
}

func FormatTime(minutes int16) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

func TrainingTypeLabel(t model.TrainingType) string {
	switch t {
	case model.TrainingTypeGi:
		return "🥋 Gi"
	case model.TrainingTypeNoGi:
		return "🥷 No-Gi"
	case model.TrainingTypeOpenMat:
		return "🤼 Open Mat"
	default:
		return string(t)
	}
}
