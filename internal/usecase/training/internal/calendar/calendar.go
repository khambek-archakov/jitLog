// Package calendar renders a monthly inline-keyboard calendar — nav row,
// weekday header, day grid, blank cells for out-of-month/future days —
// shared by every training scenario that needs to pick a date
// (create's DateStep, update's date-edit). Each caller owns its own
// callback-data namespace (see Callbacks) and hands it in; this package
// only knows how to lay the grid out and do the year/month math.
package calendar

import (
	"fmt"
	"strconv"
	"time"

	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// Callbacks is the caller's callback-data namespace for one calendar
// instance.
type Callbacks struct {
	// MonthPrefix is followed by a "2006-01" month when navigating.
	MonthPrefix string
	// DayPrefix is followed by a "2006-01-02" date when picking a day.
	DayPrefix string
	// Noop is the callback data for non-interactive cells (blank days,
	// the month label, a disabled "next month" arrow).
	Noop string
	// Cancel is the callback data for the Cancel row's button. Empty
	// means no Cancel row is rendered.
	Cancel string
}

var russianMonthsNominative = [...]string{
	"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
	"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
}

// Text is the calendar's header text for year/month.
func Text(year int, month time.Month) string {
	return fmt.Sprintf("📅 %s %d", russianMonthsNominative[month-1], year)
}

// Keyboard lays out a month as: nav row (‹ Month ›), a weekday header row,
// one row per calendar week, and (if cb.Cancel is set) a Cancel row. Days
// outside the month and days after today are rendered as blank,
// unclickable cells so the grid stays aligned. Picking a training date in
// the future is never allowed, so › is disabled once the current month is
// already shown.
func Keyboard(year int, month time.Month, cb Callbacks) dto.Keyboard {
	keyboard := dto.Keyboard{
		navRow(year, month, cb),
		weekdayRow(cb.Noop),
	}

	keyboard = append(keyboard, dayRows(year, month, time.Now(), cb)...)

	if cb.Cancel != "" {
		keyboard = append(keyboard, dto.Row(dto.Button{Label: "Отмена", Data: cb.Cancel}))
	}

	return keyboard
}

func navRow(year int, month time.Month, cb Callbacks) []dto.Button {
	prevYear, prevMonth := AddMonths(year, month, -1)

	nextData := cb.Noop
	if year*12+int(month) < time.Now().Year()*12+int(time.Now().Month()) {
		nextYear, nextMonth := AddMonths(year, month, 1)
		nextData = monthCallback(cb.MonthPrefix, nextYear, nextMonth)
	}

	return dto.Row(
		dto.Button{Label: "‹", Data: monthCallback(cb.MonthPrefix, prevYear, prevMonth)},
		dto.Button{Label: russianMonthsNominative[month-1], Data: cb.Noop},
		dto.Button{Label: "›", Data: nextData},
	)
}

func weekdayRow(noop string) []dto.Button {
	return dto.Row(
		dto.Button{Label: "Пн", Data: noop},
		dto.Button{Label: "Вт", Data: noop},
		dto.Button{Label: "Ср", Data: noop},
		dto.Button{Label: "Чт", Data: noop},
		dto.Button{Label: "Пт", Data: noop},
		dto.Button{Label: "Сб", Data: noop},
		dto.Button{Label: "Вс", Data: noop},
	)
}

func dayRows(year int, month time.Month, now time.Time, cb Callbacks) [][]dto.Button {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	// time.Weekday: Sunday=0..Saturday=6 — shift so Monday=0..Sunday=6.
	leading := (int(first.Weekday()) + 6) % 7
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	blank := dto.Button{Label: " ", Data: cb.Noop}

	var rows [][]dto.Button

	row := make([]dto.Button, 0, 7)
	for range leading {
		row = append(row, blank)
	}

	for day := 1; day <= daysInMonth; day++ {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		btn := blank
		if !date.After(today) {
			btn = dto.Button{Label: strconv.Itoa(day), Data: dayCallback(cb.DayPrefix, year, month, day)}
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

// AddMonths returns the year/month delta calendar months away from
// year/month.
func AddMonths(year int, month time.Month, delta int) (int, time.Month) {
	t := time.Date(year, month+time.Month(delta), 1, 0, 0, 0, 0, time.UTC)
	return t.Year(), t.Month()
}

func monthCallback(prefix string, year int, month time.Month) string {
	return fmt.Sprintf("%s%04d-%02d", prefix, year, int(month))
}

func dayCallback(prefix string, year int, month time.Month, day int) string {
	return fmt.Sprintf("%s%04d-%02d-%02d", prefix, year, int(month), day)
}

// ParseYearMonth parses a "2006-01" month token — the suffix left after
// trimming a Callbacks.MonthPrefix off incoming callback data.
func ParseYearMonth(s string) (int, time.Month, bool) {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return 0, 0, false
	}

	return t.Year(), t.Month(), true
}

// ParseDate parses a "2006-01-02" date token — the suffix left after
// trimming a Callbacks.DayPrefix off incoming callback data.
func ParseDate(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, false
	}

	return t, true
}
