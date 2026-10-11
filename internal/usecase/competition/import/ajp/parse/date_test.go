package parse_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/usecase/competition/import/ajp/parse"
)

func utcDate(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestResolveDate(t *testing.T) {
	t.Parallel()

	today := utcDate(2026, time.November, 15)
	early := utcDate(2026, time.January, 15) // well before every October/November case below

	tests := []struct {
		name        string
		dateRaw     string
		today       time.Time
		wantDate    time.Time
		wantEndDate time.Time
	}{
		{
			name:        "single day, no year, still ahead of today",
			dateRaw:     "October 10",
			today:       early,
			wantDate:    utcDate(2026, time.October, 10),
			wantEndDate: utcDate(2026, time.October, 10),
		},
		{
			name:        "same-month range, no year, still ahead of today",
			dateRaw:     "October 10 - 11",
			today:       early,
			wantDate:    utcDate(2026, time.October, 10),
			wantEndDate: utcDate(2026, time.October, 11),
		},
		{
			name:        "single day, explicit year",
			dateRaw:     "2027 January 31",
			today:       today,
			wantDate:    utcDate(2027, time.January, 31),
			wantEndDate: utcDate(2027, time.January, 31),
		},
		{
			name:        "same-month range, explicit year",
			dateRaw:     "2027 January 15 - 17",
			today:       today,
			wantDate:    utcDate(2027, time.January, 15),
			wantEndDate: utcDate(2027, time.January, 17),
		},
		{
			name:        "cross-month range, no year, still ahead of today",
			dateRaw:     "October 31 - November 2",
			today:       early,
			wantDate:    utcDate(2026, time.October, 31),
			wantEndDate: utcDate(2026, time.November, 2),
		},
		{
			name:        "cross-year range, no year, still ahead of today",
			dateRaw:     "December 30 - January 2",
			today:       today, // 2026-11-15: the Dec 2026 - Jan 2027 window hasn't happened yet
			wantDate:    utcDate(2026, time.December, 30),
			wantEndDate: utcDate(2027, time.January, 2),
		},
		{
			name:        "cross-year range, no year, already in the past rolls forward a full year",
			dateRaw:     "December 30 - January 2",
			today:       utcDate(2027, time.January, 15), // the Dec 2026 - Jan 2027 window already passed
			wantDate:    utcDate(2027, time.December, 30),
			wantEndDate: utcDate(2028, time.January, 2),
		},
		{
			name:        "no year, end date already past today rolls the whole event forward a year",
			dateRaw:     "October 10",
			today:       utcDate(2026, time.December, 1), // this October 10 already happened this year
			wantDate:    utcDate(2027, time.October, 10),
			wantEndDate: utcDate(2027, time.October, 10),
		},
		{
			name:        "no year, end date is exactly today — not yet past, keeps this year",
			dateRaw:     "November 15",
			today:       today,
			wantDate:    utcDate(2026, time.November, 15),
			wantEndDate: utcDate(2026, time.November, 15),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			date, endDate, err := parse.ResolveDate(tc.dateRaw, tc.today)
			require.NoError(t, err)
			assert.True(t, tc.wantDate.Equal(date), "date: want %s, got %s", tc.wantDate, date)
			assert.True(t, tc.wantEndDate.Equal(endDate), "endDate: want %s, got %s", tc.wantEndDate, endDate)
		})
	}
}

func TestResolveDate_tooFarAhead(t *testing.T) {
	t.Parallel()

	today := utcDate(2026, time.November, 15)

	_, _, err := parse.ResolveDate("2028 December 1", today)
	assert.ErrorIs(t, err, parse.ErrDateTooFarAhead)
}

func TestResolveDate_unrecognized(t *testing.T) {
	t.Parallel()

	today := utcDate(2026, time.November, 15)

	tests := []string{
		"",
		"sometime in October",
		"October",
		"Octobre 10",
		"2026-10-10",
		"October 10 -",
		"February 30",
		"2027 February 30",
	}

	for _, dateRaw := range tests {
		t.Run(dateRaw, func(t *testing.T) {
			t.Parallel()

			_, _, err := parse.ResolveDate(dateRaw, today)
			assert.ErrorIs(t, err, parse.ErrDateUnrecognized)
		})
	}
}
