package stats

import (
	"strconv"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	traininginfo "github.com/khambek-archakov/jitLog/internal/usecase/training/info"
)

// The "📅 Другие даты" flow picks a custom (from, to) range entirely from
// callback data — no draft row, no in-memory state (see the package doc
// for why: an old message must keep working after a restart). Every step
// re-derives everything it needs from the tapped button's own data plus
// the overall (minDate, maxDate) bounds, which the caller (UseCase)
// recomputes fresh on every tap from the user's own trainings/CreatedAt
// and usertime.Today.
const (
	callbackCalPrefix = "cal:"

	// callbackCalFromMonthPrefix is followed by a "200601" month — showing
	// (or navigating) the "От" step's calendar.
	callbackCalFromMonthPrefix = "cal:f:"
	// callbackCalFromDayPrefix is followed by a "20060102" date — picking
	// the "От" day itself.
	callbackCalFromDayPrefix = "cal:d:"
	// callbackCalToMonthPrefix is followed by "<from 20060102>:<200601
	// month>" — showing (or navigating) the "До" step's calendar.
	callbackCalToMonthPrefix = "cal:t:"
	// callbackCalToDayPrefix is followed by "<from 20060102>:<to
	// 20060102>" — picking the "До" day, landing on the confirm screen.
	callbackCalToDayPrefix = "cal:e:"
	// callbackCalApplyPrefix is followed by "<from 20060102>:<to
	// 20060102>" — "✅ Применить" on the confirm screen.
	callbackCalApplyPrefix = "cal:a:"

	// callbackCalCancel returns to the default "Неделя" period view from
	// any calendar step or the confirm screen.
	callbackCalCancel = "cal:c"
	// callbackCalNoop is every inert cell: the month/year label, the
	// weekday header, and blank grid padding. It answers the tap with no
	// visible feedback and leaves the screen untouched.
	callbackCalNoop = "cal:noop"
	// callbackCalUnavailable is a disabled day or nav arrow. Unlike
	// callbackCalNoop, it answers with a short toast explaining why.
	callbackCalUnavailable = "cal:x"
)

const calUnavailableToast = "Эта дата недоступна"

const (
	calDateLayout  = "20060102"
	calMonthLayout = "200601"
)

func formatCalDate(t time.Time) string {
	return t.Format(calDateLayout)
}

func formatCalMonth(t time.Time) string {
	return t.Format(calMonthLayout)
}

func parseCalDate(s string) (time.Time, bool) {
	t, err := time.ParseInLocation(calDateLayout, s, time.UTC)
	if err != nil {
		return time.Time{}, false
	}

	return t, true
}

func parseCalMonth(s string) (time.Time, bool) {
	t, err := time.ParseInLocation(calMonthLayout, s, time.UTC)
	if err != nil {
		return time.Time{}, false
	}

	return t, true
}

// parseCalFromMonth trims callbackCalFromMonthPrefix.
func parseCalFromMonth(data string) (time.Time, bool) {
	return parseCalMonth(strings.TrimPrefix(data, callbackCalFromMonthPrefix))
}

// parseCalFromDay trims callbackCalFromDayPrefix.
func parseCalFromDay(data string) (time.Time, bool) {
	return parseCalDate(strings.TrimPrefix(data, callbackCalFromDayPrefix))
}

// parseCalToMonth trims callbackCalToMonthPrefix and splits the "От" date
// from the displayed month.
func parseCalToMonth(data string) (from, month time.Time, ok bool) {
	return parseCalPair(strings.TrimPrefix(data, callbackCalToMonthPrefix), parseCalDate, parseCalMonth)
}

// parseCalToDay trims callbackCalToDayPrefix and splits "От" from "До".
func parseCalToDay(data string) (from, to time.Time, ok bool) {
	return parseCalPair(strings.TrimPrefix(data, callbackCalToDayPrefix), parseCalDate, parseCalDate)
}

// parseCalApply trims callbackCalApplyPrefix and splits "От" from "До".
func parseCalApply(data string) (from, to time.Time, ok bool) {
	return parseCalPair(strings.TrimPrefix(data, callbackCalApplyPrefix), parseCalDate, parseCalDate)
}

func parseCalPair(
	rest string, parseFirst, parseSecond func(string) (time.Time, bool),
) (first, second time.Time, ok bool) {
	firstStr, secondStr, found := strings.Cut(rest, ":")
	if !found {
		return time.Time{}, time.Time{}, false
	}

	first, ok = parseFirst(firstStr)
	if !ok {
		return time.Time{}, time.Time{}, false
	}

	second, ok = parseSecond(secondStr)
	if !ok {
		return time.Time{}, time.Time{}, false
	}

	return first, second, true
}

func calFromMonthCallback(month time.Time) string {
	return callbackCalFromMonthPrefix + formatCalMonth(month)
}

func calFromDayCallback(day time.Time) string {
	return callbackCalFromDayPrefix + formatCalDate(day)
}

func calToMonthCallback(from, month time.Time) string {
	return callbackCalToMonthPrefix + formatCalDate(from) + ":" + formatCalMonth(month)
}

func calToDayCallback(from, to time.Time) string {
	return callbackCalToDayPrefix + formatCalDate(from) + ":" + formatCalDate(to)
}

func calApplyCallback(from, to time.Time) string {
	return callbackCalApplyPrefix + formatCalDate(from) + ":" + formatCalDate(to)
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func addCalMonths(t time.Time, delta int) time.Time {
	return time.Date(t.Year(), t.Month()+time.Month(delta), 1, 0, 0, 0, 0, time.UTC)
}

func monthIndex(t time.Time) int {
	return t.Year()*12 + int(t.Month())
}

func calMonthLabel(month time.Time) string {
	return monthsNominative[month.Month()-1] + " " + strconv.Itoa(month.Year())
}

// calFromText is the "От" step's message text — just the prompt, the
// displayed month already shows in the nav row's own label.
func calFromText() string {
	return "Выбери начало периода"
}

// calToText is the "До" step's message text.
func calToText(from time.Time) string {
	return "Выбери конец периода\nНачало: " + traininginfo.FormatDate(from)
}

// calConfirmText is the confirm screen shown right after "До" is picked —
// distinct from the final applied stats screen's own header (rangeText).
func calConfirmText(from, to time.Time) string {
	return "Период: " + traininginfo.FormatDate(from) + " – " + traininginfo.FormatDate(to)
}

func calWeekdayRow() []dto.Button {
	return dto.Row(
		dto.Button{Label: "Пн", Data: callbackCalNoop},
		dto.Button{Label: "Вт", Data: callbackCalNoop},
		dto.Button{Label: "Ср", Data: callbackCalNoop},
		dto.Button{Label: "Чт", Data: callbackCalNoop},
		dto.Button{Label: "Пт", Data: callbackCalNoop},
		dto.Button{Label: "Сб", Data: callbackCalNoop},
		dto.Button{Label: "Вс", Data: callbackCalNoop},
	)
}

// calNavRow builds the «/‹/{Month Year}/›/» row. month is the currently
// displayed month; minDate/maxDate bound the overall selectable range —
// an arrow that would step outside it renders as a disabled "·" instead
// of moving anywhere. toCallback builds the target callback for a month
// that's still in range (calFromMonthCallback, or calToMonthCallback
// bound to its own "От" date).
func calNavRow(month, minDate, maxDate time.Time, toCallback func(time.Time) string) []dto.Button {
	minMonth, maxMonth := monthIndex(startOfMonth(minDate)), monthIndex(startOfMonth(maxDate))

	arrow := func(delta int) dto.Button {
		target := addCalMonths(month, delta)

		label := "‹"
		switch delta {
		case -12:
			label = "«"
		case 1:
			label = "›"
		case 12:
			label = "»"
		}

		if monthIndex(target) < minMonth || monthIndex(target) > maxMonth {
			return dto.Button{Label: "·", Data: callbackCalUnavailable}
		}

		return dto.Button{Label: label, Data: toCallback(target)}
	}

	return dto.Row(
		arrow(-12), arrow(-1),
		dto.Button{Label: calMonthLabel(month), Data: callbackCalNoop},
		arrow(1), arrow(12),
	)
}

// calDayRows lays the month out as full weeks (Monday-start), padding
// leading/trailing cells with inert "·" blanks so every row has 7
// buttons. A day outside [minDate, maxDate] (and, when fromFloor is set,
// before it too) renders as a disabled "·" instead of a pickable number.
func calDayRows(month, minDate, maxDate time.Time, fromFloor *time.Time, dayCallback func(time.Time) string) dto.Keyboard {
	first := startOfMonth(month)
	// time.Weekday: Sunday=0..Saturday=6 — shift so Monday=0..Sunday=6.
	leading := (int(first.Weekday()) + 6) % 7
	daysInMonth := addCalMonths(month, 1).AddDate(0, 0, -1).Day()

	blank := dto.Button{Label: "·", Data: callbackCalNoop}

	var rows dto.Keyboard

	row := make([]dto.Button, 0, 7)
	for range leading {
		row = append(row, blank)
	}

	for day := 1; day <= daysInMonth; day++ {
		date := time.Date(month.Year(), month.Month(), day, 0, 0, 0, 0, time.UTC)

		btn := dto.Button{Label: strconv.Itoa(day), Data: dayCallback(date)}
		if date.Before(minDate) || date.After(maxDate) || (fromFloor != nil && date.Before(*fromFloor)) {
			btn = dto.Button{Label: "·", Data: callbackCalUnavailable}
		}

		row = append(row, btn)

		if len(row) == 7 {
			rows = append(rows, row)
			row = make([]dto.Button, 0, 7)
		}
	}

	if len(row) > 0 {
		for len(row) < 7 {
			row = append(row, blank)
		}

		rows = append(rows, row)
	}

	return rows
}

// calFromKeyboard renders the "От" step at month.
func calFromKeyboard(month, minDate, maxDate time.Time) dto.Keyboard {
	kb := dto.Keyboard{calNavRow(month, minDate, maxDate, calFromMonthCallback), calWeekdayRow()}
	kb = append(kb, calDayRows(month, minDate, maxDate, nil, calFromDayCallback)...)
	kb = append(kb, dto.Row(dto.Button{Label: "Отмена", Data: callbackCalCancel}))

	return kb
}

// calToKeyboard renders the "До" step at month, with from already fixed.
func calToKeyboard(from, month, minDate, maxDate time.Time) dto.Keyboard {
	kb := dto.Keyboard{
		calNavRow(month, minDate, maxDate, func(m time.Time) string { return calToMonthCallback(from, m) }),
		calWeekdayRow(),
	}
	kb = append(kb, calDayRows(month, minDate, maxDate, &from, func(d time.Time) string { return calToDayCallback(from, d) })...)
	kb = append(
		kb,
		dto.Row(dto.Button{Label: "← Изменить начало", Data: calFromMonthCallback(startOfMonth(from))}),
		dto.Row(dto.Button{Label: "Отмена", Data: callbackCalCancel}),
	)

	return kb
}

// calConfirmKeyboard is the screen shown right after "До" is picked.
func calConfirmKeyboard(from, to time.Time) dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "✅ Применить", Data: calApplyCallback(from, to)}),
		dto.Row(dto.Button{Label: "← Изменить начало", Data: calFromMonthCallback(startOfMonth(from))}),
		dto.Row(dto.Button{Label: "Отмена", Data: callbackCalCancel}),
	}
}
