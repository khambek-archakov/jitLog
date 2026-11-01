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
// re-derives everything it needs from the tapped button's own data; the
// only restriction on which day/month can be picked is the fixed
// calAbsoluteMin/calAbsoluteMax window below — any real date within it
// (future, past, before the user's first training, a neighboring month)
// is always selectable.
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
	// callbackCalNoop is every inert cell or control: the weekday header,
	// a grid cell whose date falls outside [calAbsoluteMin,
	// calAbsoluteMax], and a nav arrow that would step outside that same
	// window. It answers the tap with no visible feedback (just clearing
	// the loading spinner) and leaves the screen untouched.
	callbackCalNoop = "cal:noop"
)

// calAbsoluteMin/calAbsoluteMax are the calendar's own hard edges — never
// tied to any user data (no "first training", no "today"). A grid cell or
// nav arrow that would land outside this window is inert (callbackCalNoop)
// instead of live.
var (
	calAbsoluteMin = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	calAbsoluteMax = time.Date(2100, 12, 31, 0, 0, 0, 0, time.UTC)
)

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

// calBoldMonth is the open month/year line, bold in MarkdownV2 — escaped
// defensively even though a Russian month name never actually contains a
// MarkdownV2 special character today.
func calBoldMonth(month time.Time) string {
	return "*" + escapeMarkdownV2(calMonthLabel(month)) + "*"
}

// calFromText is the "От" step's message text — the open month/year now
// shows here (bold), not in the keyboard, which lost its own label button.
func calFromText(month time.Time) string {
	return "Выбери начало периода\n" + calBoldMonth(month)
}

// calToText is the "До" step's message text. month is the step's own
// currently open month, which can drift away from from's month as the
// user pages the calendar — that's what calBoldMonth always reflects.
func calToText(from, month time.Time) string {
	return "Выбери конец периода\nНачало: " + escapeMarkdownV2(traininginfo.FormatDate(from)) + "\n" + calBoldMonth(month)
}

// calConfirmText is the confirm screen shown right after "До" is picked —
// distinct from the final applied stats screen's own header (rangeText).
// Plain text, no MarkdownV2 — unchanged from before this screen's own fix.
func calConfirmText(from, to time.Time) string {
	return "Период: " + traininginfo.FormatDate(from) + " – " + traininginfo.FormatDate(to)
}

// escapeMarkdownV2 escapes every character MarkdownV2 treats specially.
// None of this package's own content (Russian month names, FormatDate's
// output) actually contains one today, but the text this feeds into is
// parsed as MarkdownV2, so this stays correct if that ever changes.
func escapeMarkdownV2(s string) string {
	var b strings.Builder

	for _, r := range s {
		switch r {
		case '_', '*', '[', ']', '(', ')', '~', '`', '>', '#', '+', '-', '=', '|', '{', '}', '.', '!', '\\':
			b.WriteByte('\\')
		}

		b.WriteRune(r)
	}

	return b.String()
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

// calNavRow builds the «/‹/›/» row — just the four arrows now, no
// month/year label button (that moved into the message text). An arrow
// that would step outside [calAbsoluteMin, calAbsoluteMax] still shows
// its normal label, but is wired to callbackCalNoop instead of actually
// moving — "на границе нажатие игнорируется". toCallback builds the
// target callback for a month still in range (calFromMonthCallback, or
// calToMonthCallback bound to its own "От" date).
func calNavRow(month time.Time, toCallback func(time.Time) string) []dto.Button {
	minMonth, maxMonth := monthIndex(calAbsoluteMin), monthIndex(calAbsoluteMax)

	arrow := func(delta int, label string) dto.Button {
		target := addCalMonths(month, delta)

		if monthIndex(target) < minMonth || monthIndex(target) > maxMonth {
			return dto.Button{Label: label, Data: callbackCalNoop}
		}

		return dto.Button{Label: label, Data: toCallback(target)}
	}

	return dto.Row(arrow(-12, "«"), arrow(-1, "‹"), arrow(1, "›"), arrow(12, "»"))
}

// calDayLabel is the one place that decides how a grid cell shows its own
// day number — plain for the open month, parenthesized for a neighboring
// month's day (e.g. "(28)"). Swap the neighboring-month style here (e.g.
// for "·28" on a narrow screen) without touching calDayRows itself. Never
// escaped: this is always an inline-button label, which Telegram never
// runs through any ParseMode/MarkdownV2 parsing.
func calDayLabel(day, openMonth time.Time) string {
	if day.Year() == openMonth.Year() && day.Month() == openMonth.Month() {
		return strconv.Itoa(day.Day())
	}

	return "(" + strconv.Itoa(day.Day()) + ")"
}

// calGridCells is always six full weeks, so the keyboard's own height
// never changes as the user pages between months.
const calGridCells = 42

// calGridStart is the Monday on/before month's own 1st.
func calGridStart(month time.Time) time.Time {
	first := startOfMonth(month)
	// time.Weekday: Sunday=0..Saturday=6 — shift so Monday=0..Sunday=6.
	leading := (int(first.Weekday()) + 6) % 7

	return first.AddDate(0, 0, -leading)
}

// calDayRows lays month out as exactly six Monday-start weeks (42 cells),
// padding with the neighboring months' own days (see calDayLabel) rather
// than blanks. Every one of those days is pickable with the exact same
// callback shape as a day in the open month (dayCallback always takes the
// cell's own absolute date) — picking one does not re-page the grid to
// its month, by construction: it's a plain day-selection callback, same
// as any other. The sole exception is a cell whose date falls outside
// [calAbsoluteMin, calAbsoluteMax], which renders as an inert "·".
func calDayRows(month time.Time, dayCallback func(time.Time) string) dto.Keyboard {
	start := calGridStart(month)

	rows := make(dto.Keyboard, 0, calGridCells/7)
	row := make([]dto.Button, 0, 7)

	for i := range calGridCells {
		day := start.AddDate(0, 0, i)

		btn := dto.Button{Label: calDayLabel(day, month), Data: dayCallback(day)}
		if day.Before(calAbsoluteMin) || day.After(calAbsoluteMax) {
			btn = dto.Button{Label: "·", Data: callbackCalNoop}
		}

		row = append(row, btn)

		if len(row) == 7 {
			rows = append(rows, row)
			row = make([]dto.Button, 0, 7)
		}
	}

	return rows
}

// calFromKeyboard renders the "От" step at month.
func calFromKeyboard(month time.Time) dto.Keyboard {
	kb := dto.Keyboard{calNavRow(month, calFromMonthCallback), calWeekdayRow()}
	kb = append(kb, calDayRows(month, calFromDayCallback)...)
	kb = append(kb, dto.Row(dto.Button{Label: "Отмена", Data: callbackCalCancel}))

	return kb
}

// calToKeyboard renders the "До" step at month, with from already fixed.
func calToKeyboard(from, month time.Time) dto.Keyboard {
	kb := dto.Keyboard{
		calNavRow(month, func(m time.Time) string { return calToMonthCallback(from, m) }),
		calWeekdayRow(),
	}
	kb = append(kb, calDayRows(month, func(d time.Time) string { return calToDayCallback(from, d) })...)
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
