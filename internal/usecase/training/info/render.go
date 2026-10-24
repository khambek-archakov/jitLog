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

// ConfirmationText/ConfirmationSummary are a compact, single-line summary
// — deliberately not Card, which is the full multi-line view shown
// everywhere else (history, view, edit). Right after creating a training
// (manually, or via the post-session reminder's own "✅ Записать" button)
// there's nothing to editorialize yet, so a dense one-liner plus
// ConfirmationKeyboard's quick-action buttons says more with less.
func ConfirmationText(t *model.Training) string {
	return "✅ Тренировка сохранена!\n" + ConfirmationSummary(t)
}

func ConfirmationSummary(t *model.Training) string {
	return fmt.Sprintf("📅 %s · %s · ⏱ %s", FormatDateShort(t.Date), TrainingTypeLabel(t.TrainingType), FormatDuration(t.DurationMinutes))
}

// ConfirmationKeyboard is the confirmation screen's own quick-action
// buttons.
func ConfirmationKeyboard(trainingID int64) dto.Keyboard {
	editPrefix := fmt.Sprintf("%s%d:", callbackTrainingEditPrefix, trainingID)

	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "➕ Схватки", Data: editPrefix + "rounds"},
			dto.Button{Label: "📝 Заметка", Data: editPrefix + "notes"},
		),
		dto.Row(
			dto.Button{Label: "✏️ Изменить", Data: fmt.Sprintf("%s%d", callbackTrainingEditPrefix, trainingID)},
			dto.Button{Label: "🗑 Удалить", Data: fmt.Sprintf("%s%d", callbackTrainingDeletePrefix, trainingID)},
		),
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

// roundsWord picks the right Russian plural form of "схватка" for n.
func roundsWord(n int16) string {
	if n%100 >= 11 && n%100 <= 14 {
		return "схваток"
	}

	switch n % 10 {
	case 1:
		return "схватка"
	case 2, 3, 4:
		return "схватки"
	default:
		return "схваток"
	}
}
