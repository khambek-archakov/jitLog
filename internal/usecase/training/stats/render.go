package stats

import (
	"fmt"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackMenuAddTraining and callbackMenuBack mirror
// internal/usecase/internal/menu's and internal/usecase/onboarding/steps'
// own private constants.
const (
	callbackMenuAddTraining = "menu:add_training"
	callbackMenuBack        = "menu:back"
)

func emptyStateText() string {
	return "📊 Статистика\n\nПока нет ни одной тренировки — добавь первую, и здесь появится твой прогресс."
}

func emptyStateKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "➕ Добавить тренировку", Data: callbackMenuAddTraining}),
		dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack}),
	}
}

func statsText(p period, now time.Time, c counts, streakWeeks int) string {
	header := "📊 Статистика · " + periodLabel(p)
	if r := periodRangeText(p, now); r != "" {
		header += " (" + r + ")"
	}

	if c.Total == 0 {
		text := header + "\n\nВ этот период тренировок не было."
		if s := streakLine(streakWeeks); s != "" {
			text += "\n\n" + s
		}

		return text
	}

	var b strings.Builder

	fmt.Fprintf(&b, "%s\n\nТренировок: %d\nЧасов на мате: %d\n\nПо типу\n", header, c.Total, c.hours())
	b.WriteString(typeLine("🥋 Gi", c.GiCount, c))
	b.WriteString(typeLine("🥷 No-Gi", c.NoGiCount, c))
	b.WriteString(typeLine("🤼 Open Mat", c.OpenMatCount, c))

	if s := streakLine(streakWeeks); s != "" {
		b.WriteString("\n")
		b.WriteString(s)
	}

	return b.String()
}

func typeLine(label string, n int, c counts) string {
	percent := c.percent(n)
	return fmt.Sprintf("%s %s %d%%\n", label, bar(percent), percent)
}

func bar(percent int) string {
	const width = 10

	filled := (percent*width + 50) / 100
	if filled > width {
		filled = width
	}
	if filled < 0 {
		filled = 0
	}

	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func streakLine(weeks int) string {
	if weeks == 0 {
		return ""
	}

	return fmt.Sprintf("🔥 Серия — %d %s подряд", weeks, weeksWord(weeks))
}

// weeksWord picks the right Russian plural form of "неделя" for n.
func weeksWord(n int) string {
	if n%100 >= 11 && n%100 <= 14 {
		return "недель"
	}

	switch n % 10 {
	case 1:
		return "неделя"
	case 2, 3, 4:
		return "недели"
	default:
		return "недель"
	}
}

func statsKeyboard(selected period) dto.Keyboard {
	return dto.Keyboard{
		dto.Row(
			periodButton("Неделя", periodWeek, selected),
			periodButton("Месяц", periodMonth, selected),
			periodButton("Год", periodYear, selected),
			periodButton("Всё время", periodAll, selected),
		),
		dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack}),
	}
}

func periodButton(label string, p, selected period) dto.Button {
	if p == selected {
		label = "• " + label + " •"
	}

	return dto.Button{Label: label, Data: callbackStatsPeriodPrefix + string(p)}
}
