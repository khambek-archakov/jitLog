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

// Body formats a slot's day/time/duration/type — shared by the card view
// and create's confirmation screen.
func Body(s *model.ScheduleSlot) string {
	return fmt.Sprintf(
		"%s — %s · %s · %s",
		DayLabel(s.DayOfWeek), FormatTime(s.TimeMinutes), FormatDuration(s.DurationMinutes), TrainingTypeLabel(s.TrainingType),
	)
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

// FormatDuration renders a slot's own duration_min as "N минут" with the
// correct Russian plural form — same n%10/n%100 pattern as every other
// count pluralizer in this codebase (training/info.minutesWord,
// training/stats.weeksWord, profile.trainingWord).
func FormatDuration(minutes int16) string {
	return fmt.Sprintf("%d %s", minutes, minutesWord(minutes))
}

func minutesWord(n int16) string {
	if n%100 >= 11 && n%100 <= 14 {
		return "минут"
	}

	switch n % 10 {
	case 1:
		return "минута"
	case 2, 3, 4:
		return "минуты"
	default:
		return "минут"
	}
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
