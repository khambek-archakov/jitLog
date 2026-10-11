package parse

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ErrDateUnrecognized means DateRaw didn't match any of AJP's own
// date-string shapes at all — the caller skips that one card and reports
// why; the import itself never fails because of it.
var ErrDateUnrecognized = errors.New("date format not recognized")

// ErrDateTooFarAhead means the resolved date is more than 18 months past
// today — AJP's own calendar never plans that far out, so this is almost
// certainly a parsing mistake rather than a real event; the card is
// skipped rather than trusted blindly.
var ErrDateTooFarAhead = errors.New("date more than 18 months ahead")

var months = map[string]time.Month{
	"January": time.January, "February": time.February, "March": time.March,
	"April": time.April, "May": time.May, "June": time.June,
	"July": time.July, "August": time.August, "September": time.September,
	"October": time.October, "November": time.November, "December": time.December,
}

// dateRe covers every shape seen on the page: "October 10",
// "October 10 - 11", "October 31 - November 2", "2027 January 31",
// "2027 January 15 - 17", "December 30 - January 2".
var dateRe = regexp.MustCompile(`^(?:(\d{4})\s+)?([A-Za-z]+)\s+(\d{1,2})(?:\s*-\s*(?:([A-Za-z]+)\s+)?(\d{1,2}))?$`)

// ResolveDate turns one of AJP's own date-string shapes into a concrete
// (date, endDate) pair. today is passed in explicitly (UTC, midnight)
// rather than read from time.Now() so every rule here — including the
// year-rollover ones — stays deterministic and table-testable.
//
// When dateRaw carries no year, today's own year is tried first; if the
// event's end date would already be in the past, both ends roll forward
// one year instead (this is what keeps a December 30 - January 2 event
// from ever landing in the wrong year, whichever side of New Year's today
// happens to fall on). An explicit year is always trusted as given.
func ResolveDate(dateRaw string, today time.Time) (date, endDate time.Time, err error) {
	trimmed := strings.TrimSpace(dateRaw)

	m := dateRe.FindStringSubmatch(trimmed)
	if m == nil {
		return time.Time{}, time.Time{}, fmt.Errorf("%q: %w", dateRaw, ErrDateUnrecognized)
	}

	yearRaw, month1Raw, day1Raw, month2Raw, day2Raw := m[1], m[2], m[3], m[4], m[5]

	month1, ok := months[month1Raw]
	if !ok {
		return time.Time{}, time.Time{}, fmt.Errorf("%q: %w", dateRaw, ErrDateUnrecognized)
	}

	day1, err := strconv.Atoi(day1Raw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("%q: %w", dateRaw, ErrDateUnrecognized)
	}

	month2 := month1

	if month2Raw != "" {
		month2, ok = months[month2Raw]
		if !ok {
			return time.Time{}, time.Time{}, fmt.Errorf("%q: %w", dateRaw, ErrDateUnrecognized)
		}
	}

	day2 := day1

	if day2Raw != "" {
		day2, err = strconv.Atoi(day2Raw)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("%q: %w", dateRaw, ErrDateUnrecognized)
		}
	}

	var startYear, endYear int

	if yearRaw != "" {
		startYear, err = strconv.Atoi(yearRaw)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("%q: %w", dateRaw, ErrDateUnrecognized)
		}

		endYear = startYear
		if month2 < month1 {
			endYear++
		}
	} else {
		startYear = today.Year()

		endYear = startYear
		if month2 < month1 {
			endYear++
		}

		if time.Date(endYear, month2, day2, 0, 0, 0, 0, time.UTC).Before(today) {
			startYear++
			endYear++
		}
	}

	date = time.Date(startYear, month1, day1, 0, 0, 0, 0, time.UTC)
	endDate = time.Date(endYear, month2, day2, 0, 0, 0, 0, time.UTC)

	// time.Date silently normalizes an out-of-range day (e.g. "February
	// 30") instead of failing — catch that here rather than trust a
	// date that quietly rolled into the next month.
	if date.Day() != day1 || date.Month() != month1 || endDate.Day() != day2 || endDate.Month() != month2 {
		return time.Time{}, time.Time{}, fmt.Errorf("%q: %w", dateRaw, ErrDateUnrecognized)
	}

	if date.After(today.AddDate(0, 18, 0)) {
		return time.Time{}, time.Time{}, fmt.Errorf("%s: %w", date.Format("2006-01-02"), ErrDateTooFarAhead)
	}

	return date, endDate, nil
}
