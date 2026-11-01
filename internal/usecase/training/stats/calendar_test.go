package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func calDate(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		panic(err)
	}

	return t
}

// TestCalDayRows_October2026 guards the spec's own worked example: October
// 2026 starts on a Thursday and has 31 days.
func TestCalDayRows_October2026(t *testing.T) {
	t.Parallel()

	month := calDate("2026-10-01")
	minDate, maxDate := calDate("2026-01-01"), calDate("2026-12-31")

	rows := calDayRows(month, minDate, maxDate, nil, calFromDayCallback)

	require.Len(t, rows, 5)
	require.Len(t, rows[0], 7)

	// 3 leading blanks (Mon, Tue, Wed), then Thursday the 1st.
	assert.Equal(t, "·", rows[0][0].Label)
	assert.Equal(t, "·", rows[0][1].Label)
	assert.Equal(t, "·", rows[0][2].Label)
	assert.Equal(t, "1", rows[0][3].Label)

	var total int

	for _, row := range rows {
		for _, btn := range row {
			if btn.Label != "·" {
				total++
			}
		}
	}

	assert.Equal(t, 31, total)

	// Trailing blank after the 31st (a Saturday) fills the last row to 7.
	last := rows[len(rows)-1]
	assert.Equal(t, "31", last[len(last)-2].Label)
	assert.Equal(t, "·", last[len(last)-1].Label)
}

func TestCalDayRows_FebruaryLeapAndNonLeap(t *testing.T) {
	t.Parallel()

	minDate, maxDate := calDate("2000-01-01"), calDate("2030-01-01")

	leapDays := countCalDays(calDayRows(calDate("2028-02-01"), minDate, maxDate, nil, calFromDayCallback))
	assert.Equal(t, 29, leapDays)

	nonLeapDays := countCalDays(calDayRows(calDate("2026-02-01"), minDate, maxDate, nil, calFromDayCallback))
	assert.Equal(t, 28, nonLeapDays)
}

func countCalDays(rows dto.Keyboard) int {
	n := 0

	for _, row := range rows {
		for _, btn := range row {
			if btn.Label != "·" {
				n++
			}
		}
	}

	return n
}

func TestAddCalMonths_CrossesYearBoundary(t *testing.T) {
	t.Parallel()

	dec := calDate("2026-12-15")

	jan := addCalMonths(dec, 1)
	assert.Equal(t, "2027-01-01", formatCalDateForTest(jan))

	nov := addCalMonths(dec, -1)
	assert.Equal(t, "2026-11-01", formatCalDateForTest(nov))

	nextYear := addCalMonths(dec, 12)
	assert.Equal(t, "2027-12-01", formatCalDateForTest(nextYear))

	prevYear := addCalMonths(dec, -12)
	assert.Equal(t, "2025-12-01", formatCalDateForTest(prevYear))
}

func formatCalDateForTest(t time.Time) string {
	return t.Format("2006-01-02")
}

func TestCalNavRow_ClampsAtFirstAndLastAvailableMonth(t *testing.T) {
	t.Parallel()

	t.Run("at the earliest month, back arrows are disabled", func(t *testing.T) {
		t.Parallel()

		minDate, maxDate := calDate("2026-09-10"), calDate("2026-11-20")
		row := calNavRow(calDate("2026-09-01"), minDate, maxDate, calFromMonthCallback)

		assert.Equal(t, callbackCalUnavailable, row[0].Data)    // «
		assert.Equal(t, callbackCalUnavailable, row[1].Data)    // ‹
		assert.NotEqual(t, callbackCalUnavailable, row[3].Data) // ›
	})

	t.Run("at the latest month, forward arrows are disabled", func(t *testing.T) {
		t.Parallel()

		minDate, maxDate := calDate("2026-09-10"), calDate("2026-11-20")
		row := calNavRow(calDate("2026-11-01"), minDate, maxDate, calFromMonthCallback)

		assert.NotEqual(t, callbackCalUnavailable, row[1].Data) // ‹
		assert.Equal(t, callbackCalUnavailable, row[3].Data)    // ›
		assert.Equal(t, callbackCalUnavailable, row[4].Data)    // »
	})

	t.Run("in the middle, every arrow is live", func(t *testing.T) {
		t.Parallel()

		// A full year clear on either side, so even the «/» year jumps
		// stay in range.
		minDate, maxDate := calDate("2025-01-01"), calDate("2027-12-31")
		row := calNavRow(calDate("2026-10-01"), minDate, maxDate, calFromMonthCallback)

		for _, i := range []int{0, 1, 3, 4} {
			assert.NotEqual(t, callbackCalUnavailable, row[i].Data)
		}
	})
}

func TestCalDayRows_Unavailable(t *testing.T) {
	t.Parallel()

	minDate, maxDate := calDate("2026-10-05"), calDate("2026-10-20")

	t.Run("before the first available day is unavailable", func(t *testing.T) {
		t.Parallel()

		rows := calDayRows(calDate("2026-10-01"), minDate, maxDate, nil, calFromDayCallback)
		btn := dayButton(rows, 3)
		assert.Equal(t, "·", btn.Label)
		assert.Equal(t, callbackCalUnavailable, btn.Data)
	})

	t.Run("after today (maxDate) is unavailable", func(t *testing.T) {
		t.Parallel()

		rows := calDayRows(calDate("2026-10-01"), minDate, maxDate, nil, calFromDayCallback)
		btn := dayButton(rows, 25)
		assert.Equal(t, "·", btn.Label)
		assert.Equal(t, callbackCalUnavailable, btn.Data)
	})

	t.Run("on the То step, before the chosen От is unavailable", func(t *testing.T) {
		t.Parallel()

		from := calDate("2026-10-12")
		rows := calDayRows(calDate("2026-10-01"), minDate, maxDate, &from, func(d time.Time) string {
			return calToDayCallback(from, d)
		})

		assert.Equal(t, callbackCalUnavailable, dayButton(rows, 10).Data)    // before from, still ≥ minDate
		assert.NotEqual(t, callbackCalUnavailable, dayButton(rows, 12).Data) // from itself is pickable
		assert.NotEqual(t, callbackCalUnavailable, dayButton(rows, 15).Data)
	})

	t.Run("available days are pickable", func(t *testing.T) {
		t.Parallel()

		rows := calDayRows(calDate("2026-10-01"), minDate, maxDate, nil, calFromDayCallback)
		btn := dayButton(rows, 10)
		assert.Equal(t, "10", btn.Label)
		assert.Equal(t, "cal:d:20261010", btn.Data)
	})
}

// dayButton finds day's own cell by position: leading blanks + (day-1).
// Every test using it builds the grid at October 2026, which starts on a
// Thursday (3 leading blanks).
func dayButton(rows dto.Keyboard, day int) dto.Button {
	const leading = 3

	idx := leading + day - 1
	rowIdx, col := idx/7, idx%7

	return rows[rowIdx][col]
}

func TestCalFromDayCallback_RoundTrips(t *testing.T) {
	t.Parallel()

	day := calDate("2026-10-15")

	data := calFromDayCallback(day)
	assert.Equal(t, "cal:d:20261015", data)

	got, ok := parseCalFromDay(data)
	require.True(t, ok)
	assert.True(t, got.Equal(day))
}

func TestCalToDayCallback_RoundTrips(t *testing.T) {
	t.Parallel()

	from, to := calDate("2026-09-01"), calDate("2026-10-15")

	data := calToDayCallback(from, to)
	assert.Equal(t, "cal:e:20260901:20261015", data)

	gotFrom, gotTo, ok := parseCalToDay(data)
	require.True(t, ok)
	assert.True(t, gotFrom.Equal(from))
	assert.True(t, gotTo.Equal(to))
}

func TestCalToMonthCallback_RoundTrips(t *testing.T) {
	t.Parallel()

	from, month := calDate("2026-09-01"), calDate("2026-10-01")

	data := calToMonthCallback(from, month)
	assert.Equal(t, "cal:t:20260901:202610", data)

	gotFrom, gotMonth, ok := parseCalToMonth(data)
	require.True(t, ok)
	assert.True(t, gotFrom.Equal(from))
	assert.True(t, gotMonth.Equal(month))
}

func TestCalApplyCallback_RoundTrips(t *testing.T) {
	t.Parallel()

	from, to := calDate("2026-09-01"), calDate("2026-10-15")

	data := calApplyCallback(from, to)
	assert.Equal(t, "cal:a:20260901:20261015", data)

	gotFrom, gotTo, ok := parseCalApply(data)
	require.True(t, ok)
	assert.True(t, gotFrom.Equal(from))
	assert.True(t, gotTo.Equal(to))
}

// TestCalCallbacks_FitWithin64Bytes guards the spec's own hard limit —
// built with the longest realistic dates (4-digit years never actually
// exceed this, but the check costs nothing to keep honest).
func TestCalCallbacks_FitWithin64Bytes(t *testing.T) {
	t.Parallel()

	from, to := calDate("2026-09-01"), calDate("2026-10-15")
	month := calDate("2026-10-01")

	callbacks := []string{
		calFromMonthCallback(month),
		calFromDayCallback(to),
		calToMonthCallback(from, month),
		calToDayCallback(from, to),
		calApplyCallback(from, to),
		callbackCalCancel,
		callbackCalNoop,
		callbackCalUnavailable,
	}

	for _, cb := range callbacks {
		assert.LessOrEqual(t, len(cb), 64, "callback %q exceeds 64 bytes", cb)
	}
}

func TestMinAvailableDate(t *testing.T) {
	t.Parallel()

	t.Run("no trainings falls back to the signup date", func(t *testing.T) {
		t.Parallel()

		u := &model.User{CreatedAt: time.Date(2026, 5, 10, 14, 30, 0, 0, time.UTC)}

		assert.True(t, minAvailableDate(u, nil).Equal(calDate("2026-05-10")))
	})

	t.Run("the earliest training wins over the signup date", func(t *testing.T) {
		t.Parallel()

		u := &model.User{CreatedAt: time.Date(2026, 5, 10, 14, 30, 0, 0, time.UTC)}

		all := []*model.Training{
			{Date: calDate("2026-06-01")},
			{Date: calDate("2026-01-15")}, // earliest, deliberately not first in the slice
		}

		assert.True(t, minAvailableDate(u, all).Equal(calDate("2026-01-15")))
	})

	t.Run("a training logged before signup still wins", func(t *testing.T) {
		t.Parallel()

		u := &model.User{CreatedAt: time.Date(2026, 5, 10, 14, 30, 0, 0, time.UTC)}

		all := []*model.Training{{Date: calDate("2026-04-01")}}

		assert.True(t, minAvailableDate(u, all).Equal(calDate("2026-04-01")))
	})
}
