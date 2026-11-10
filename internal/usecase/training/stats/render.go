package stats

import (
	"fmt"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	traininginfo "github.com/khambek-archakov/jitLog/internal/usecase/training/info"
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

// beltOrder is belt_promotion's natural progression — breakdown rows
// follow this order rather than, say, first-promoted order.
var beltOrder = [...]model.Belt{
	model.BeltWhite, model.BeltBlue, model.BeltPurple, model.BeltBrown, model.BeltBlack,
}

func beltsText(c beltCounts, promotions []*model.BeltPromotion, streakWeeks int) string {
	header := "📊 Статистика · По поясам"

	if c.Total == 0 {
		text := header + "\n\nТренировок пока нет."
		if s := streakLine(streakWeeks); s != "" {
			text += "\n\n" + s
		}

		return text
	}

	var b strings.Builder

	fmt.Fprintf(
		&b, "%s\n\n🥋 Тренировок: %d\n⏱ Время на мате: %s\n\nПо поясам\n", header, c.Total, FormatDuration(c.TotalMinutes),
	)

	held := heldBelts(promotions)
	for _, belt := range beltOrder {
		if !held[belt] {
			continue
		}

		b.WriteString(beltLine(belt, c))
	}

	if s := streakLine(streakWeeks); s != "" {
		b.WriteString("\n")
		b.WriteString(s)
	}

	return b.String()
}

func heldBelts(promotions []*model.BeltPromotion) map[model.Belt]bool {
	held := make(map[model.Belt]bool, len(promotions))
	for _, p := range promotions {
		held[p.Belt] = true
	}

	return held
}

func beltLine(belt model.Belt, c beltCounts) string {
	percent := c.percent(belt)
	return fmt.Sprintf("%s %s %d%%\n", beltLabel(belt), bar(percent), percent)
}

func beltLabel(b model.Belt) string {
	switch b {
	case model.BeltWhite:
		return "⚪ Белый"
	case model.BeltBlue:
		return "🔵 Синий"
	case model.BeltPurple:
		return "🟣 Пурпурный"
	case model.BeltBrown:
		return "🟤 Коричневый"
	case model.BeltBlack:
		return "⚫ Чёрный"
	default:
		return "—"
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

	fmt.Fprintf(&b, "%s\n\n🥋 Тренировок: %d\n⏱ Время на мате: %s\n", header, c.Total, FormatDuration(c.TotalMinutes))

	if c.TotalRounds > 0 {
		fmt.Fprintf(&b, "🔄 Схваток: %d\n", c.TotalRounds)
	}

	b.WriteString("\nПо типу\n")
	b.WriteString(typeLine("🥋 Gi", c.GiCount, c))
	b.WriteString(typeLine("🥷 No-Gi", c.NoGiCount, c))
	b.WriteString(typeLine("🤼 Open Mat", c.OpenMatCount, c))

	if s := streakLine(streakWeeks); s != "" {
		b.WriteString("\n")
		b.WriteString(s)
	}

	return b.String()
}

// FormatDuration renders total minutes as "3 ч" when they divide evenly
// into hours, "45 мин" under an hour, or "2 ч 45 мин" otherwise — shared
// with internal/usecase/profile's own belt-promotion congratulation, which
// is why this one's exported.
func FormatDuration(totalMinutes int32) string {
	hours := totalMinutes / 60
	minutes := totalMinutes % 60

	switch {
	case hours == 0:
		return fmt.Sprintf("%d мин", minutes)
	case minutes == 0:
		return fmt.Sprintf("%d ч", hours)
	default:
		return fmt.Sprintf("%d ч %d мин", hours, minutes)
	}
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

// statsKeyboard builds the "За период" screen's keyboard. showModeSwitch
// is true once the user has ≥2 recorded belts — otherwise "По поясам"
// would always be a 100%-one-belt no-op, so it just isn't offered.
// showShare is true only for the Month and Year tabs with ≥1 training
// in that period — see the share feature's own spec for why it's gated
// this narrowly. now is only needed to seed "📅 Другие даты" — it always
// opens the calendar on the current month.
func statsKeyboard(selected period, showModeSwitch, showShare bool, now time.Time) dto.Keyboard {
	kb := dto.Keyboard{periodRow(selected), otherDatesRow(now)}

	if showShare {
		kb = append(kb, dto.Row(dto.Button{Label: "↗️ Поделиться", Data: shareCallback(selected)}))
	}

	if showModeSwitch {
		kb = append(kb, modeSwitchRow(false))
	}

	kb = append(kb, dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack}))

	return kb
}

// beltsKeyboard builds the "По поясам" screen's keyboard — no period
// tabs here at all, see the discussion behind this: belt and period don't
// combine, they're two independent views.
func beltsKeyboard() dto.Keyboard {
	return dto.Keyboard{
		modeSwitchRow(true),
		dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack}),
	}
}

// modeSwitchRow is shared by both screens — activeIsBelts marks whichever
// side is currently showing. "За период" always lands back on the default
// week tab; there's no session to remember a prior period selection in.
func modeSwitchRow(activeIsBelts bool) []dto.Button {
	periodLabel, beltsLabel := "За период", "По поясам"

	if activeIsBelts {
		beltsLabel = "• " + beltsLabel + " •"
	} else {
		periodLabel = "• " + periodLabel + " •"
	}

	return dto.Row(
		dto.Button{Label: periodLabel, Data: callbackStatsPeriodPrefix + string(periodWeek)},
		dto.Button{Label: beltsLabel, Data: callbackStatsBelts},
	)
}

func periodButton(label string, p, selected period) dto.Button {
	if p == selected {
		label = "• " + label + " •"
	}

	return dto.Button{Label: label, Data: callbackStatsPeriodPrefix + string(p)}
}

// periodRow is the four period tabs, shared by statsKeyboard and
// rangeKeyboard — passing a selected value that's none of the four
// (rangeKeyboard's own case, since a custom range is never literally "one
// of the tabs") just renders them all unmarked.
func periodRow(selected period) []dto.Button {
	return dto.Row(
		periodButton("Неделя", periodWeek, selected),
		periodButton("Месяц", periodMonth, selected),
		periodButton("Год", periodYear, selected),
		periodButton("Всё время", periodAll, selected),
	)
}

// otherDatesRow opens the calendar's "От" step at now's own month.
func otherDatesRow(now time.Time) []dto.Button {
	return dto.Row(dto.Button{Label: "📅 Другие даты", Data: calFromMonthCallback(startOfMonth(now))})
}

// rangeText is the applied custom-range screen's own header — distinct
// from calConfirmText, which only shows the lighter "Период: …" preview
// before "Применить" is tapped. No streak line here (see the feature's
// own spec: a custom range doesn't carry the weekly-streak concept).
func rangeText(from, to time.Time, c counts) string {
	header := "📊 Статистика · " + traininginfo.FormatDate(from) + " – " + traininginfo.FormatDate(to)

	if c.Total == 0 {
		return header + "\n\nВ этом периоде нет тренировок."
	}

	var b strings.Builder

	fmt.Fprintf(&b, "%s\n\n🥋 Тренировок: %d\n⏱ Время на мате: %s\n", header, c.Total, FormatDuration(c.TotalMinutes))

	if c.TotalRounds > 0 {
		fmt.Fprintf(&b, "🔄 Схваток: %d\n", c.TotalRounds)
	}

	b.WriteString("\nПо типу\n")
	b.WriteString(typeLine("🥋 Gi", c.GiCount, c))
	b.WriteString(typeLine("🥷 No-Gi", c.NoGiCount, c))
	b.WriteString(typeLine("🤼 Open Mat", c.OpenMatCount, c))

	return b.String()
}

// rangeKeyboard is the applied custom-range screen's own keyboard — the
// same period tabs (none marked selected) and "📅 Другие даты" row as
// statsKeyboard, but no mode switch (belts-by-range isn't offered, see
// the spec) and no "selected" period.
func rangeKeyboard(now time.Time) dto.Keyboard {
	return dto.Keyboard{
		periodRow(""),
		otherDatesRow(now),
		dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack}),
	}
}
