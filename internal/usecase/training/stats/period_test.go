package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		data string
		want period
		ok   bool
	}{
		{data: "stats:period:week", want: periodWeek, ok: true},
		{data: "stats:period:month", want: periodMonth, ok: true},
		{data: "stats:period:year", want: periodYear, ok: true},
		{data: "stats:period:all", want: periodAll, ok: true},
		{data: "stats:period:bogus", ok: false},
		{data: "junk", ok: false},
	}

	for _, tc := range tests {
		t.Run(tc.data, func(t *testing.T) {
			t.Parallel()

			p, ok := parsePeriod(tc.data)

			require.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.want, p)
			}
		})
	}
}

func TestMondayOf(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		date string
		want string
	}{
		{name: "a Monday maps to itself", date: "2026-03-16", want: "2026-03-16"},
		{name: "a Wednesday maps back to Monday", date: "2026-03-18", want: "2026-03-16"},
		{name: "a Sunday maps back to the same week's Monday", date: "2026-03-22", want: "2026-03-16"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			d, err := time.Parse("2006-01-02", tc.date)
			require.NoError(t, err)

			assert.Equal(t, tc.want, mondayOf(d).Format("2006-01-02"))
		})
	}
}

func TestPeriodStart(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	assert.Equal(t, "2026-03-16", periodStart(periodWeek, now).Format("2006-01-02"))
	assert.Equal(t, "2026-03-01", periodStart(periodMonth, now).Format("2006-01-02"))
	assert.Equal(t, "2026-01-01", periodStart(periodYear, now).Format("2006-01-02"))
	assert.True(t, periodStart(periodAll, now).IsZero())
}

// TestPeriodStart_IgnoresNowsLocation guards against the regression where
// periodStart/mondayOf were built using now.Location() instead of
// time.UTC: a server running in any non-UTC timezone would then compute a
// boundary offset from the training dates coming back from the repository
// (always UTC-located, see internal/repository/training), wrongly
// excluding trainings from the current period near a day boundary.
func TestPeriodStart_IgnoresNowsLocation(t *testing.T) {
	t.Parallel()

	utc := time.Date(2026, 3, 18, 20, 0, 0, 0, time.UTC)

	// A timezone well behind UTC: same wall-clock date (2026-03-18), but a
	// much earlier absolute instant than the UTC equivalent above.
	behindUTC := time.FixedZone("UTC-8", -8*3600)
	local := time.Date(2026, 3, 18, 12, 0, 0, 0, behindUTC)

	for _, p := range []period{periodWeek, periodMonth, periodYear} {
		assert.Equal(
			t, periodStart(p, utc).Format("2006-01-02"), periodStart(p, local).Format("2006-01-02"),
			"period %s should resolve to the same calendar boundary regardless of now's location", p,
		)
		assert.True(t, periodStart(p, local).Location() == time.UTC, "period %s boundary must be UTC-located", p)
	}
}

func TestPeriodRangeText(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	assert.Equal(t, "16–22 марта", periodRangeText(periodWeek, now))
	assert.Equal(t, "Март 2026", periodRangeText(periodMonth, now))
	assert.Equal(t, "2026", periodRangeText(periodYear, now))
	assert.Equal(t, "", periodRangeText(periodAll, now))
}

func TestWeekRangeText_CrossesMonthBoundary(t *testing.T) {
	t.Parallel()

	monday, err := time.Parse("2006-01-02", "2026-02-23")
	require.NoError(t, err)
	sunday := monday.AddDate(0, 0, 6)

	assert.Equal(t, "23 февраля – 1 марта", weekRangeText(monday, sunday))
}
