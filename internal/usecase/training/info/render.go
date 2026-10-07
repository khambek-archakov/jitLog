package info

import (
	"fmt"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	callbackTrainingViewPrefix   = "training:view:"
	callbackTrainingEditPrefix   = "training:edit:"
	callbackTrainingDeletePrefix = "training:delete:"
	// callbackHistoryFirstPage mirrors history's own first-page callback —
	// "← Назад" on a card just re-shows the list from the top.
	callbackHistoryFirstPage = "training:history:page:0"
)

// Card is the full single-training view: a headline plus Body.
func Card(t *model.Training) string {
	return "Тренировка\n\n" + Body(t)
}

// Body formats a training's date/type/duration/notes — shared by the card
// view and create's "saved" confirmation screen.
func Body(t *model.Training) string {
	body := fmt.Sprintf(
		"📅 %s\n%s\n⏱ %s",
		FormatDate(t.Date), TrainingTypeLabel(t.TrainingType), FormatDuration(t.DurationMinutes),
	)

	if t.Rounds != nil {
		body += "\n🔄 " + FormatRounds(*t.Rounds)
	}

	if t.Notes != nil && *t.Notes != "" {
		body += "\n\n📝 " + *t.Notes
	}

	return body
}

// Keyboard is the card's own layout: Изменить and Удалить each on their own
// row, plus a row back to the history list.
func Keyboard(trainingID int64) dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "✏️ Изменить", Data: fmt.Sprintf("%s%d", callbackTrainingEditPrefix, trainingID)}),
		dto.Row(dto.Button{Label: "🗑 Удалить", Data: fmt.Sprintf("%s%d", callbackTrainingDeletePrefix, trainingID)}),
		dto.Row(dto.Button{Label: "← Назад", Data: callbackHistoryFirstPage}),
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

var russianMonthsGenitive = [...]string{
	"января", "февраля", "марта", "апреля", "мая", "июня",
	"июля", "августа", "сентября", "октября", "ноября", "декабря",
}

func FormatDate(d time.Time) string {
	return fmt.Sprintf("%d %s %d", d.Day(), russianMonthsGenitive[d.Month()-1], d.Year())
}

// FormatDateShort omits the year — for the create wizard's compact
// just-saved confirmation, where "today, roughly" is all that's needed.
func FormatDateShort(d time.Time) string {
	return fmt.Sprintf("%d %s", d.Day(), russianMonthsGenitive[d.Month()-1])
}

func FormatDuration(minutes int32) string {
	return fmt.Sprintf("%d %s", minutes, minutesWord(minutes))
}

// minutesWord picks the right Russian plural form of "минута" for n.
func minutesWord(n int32) string {
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

func FormatRounds(n int16) string {
	return fmt.Sprintf("%d %s", n, roundsWord(n))
}

// roundsWord picks the right Russian plural form of "раунд" for n.
func roundsWord(n int16) string {
	if n%100 >= 11 && n%100 <= 14 {
		return "раундов"
	}

	switch n % 10 {
	case 1:
		return "раунд"
	case 2, 3, 4:
		return "раунда"
	default:
		return "раундов"
	}
}
