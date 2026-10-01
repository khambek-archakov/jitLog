package calendar_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/usecase/training/internal/calendar"
)

func TestText(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "📅 Март 2026", calendar.Text(2026, time.March))
}

func TestKeyboard(t *testing.T) {
	t.Parallel()

	cb := calendar.Callbacks{
		MonthPrefix: "cal:",
		DayPrefix:   "pick:",
		Noop:        "noop",
		Cancel:      "cancel",
	}

	t.Run("a fully past month renders every day as pickable", func(t *testing.T) {
		t.Parallel()

		kb := calendar.Keyboard(2020, time.January, cb)

		require.GreaterOrEqual(t, len(kb), 4)
		require.Len(t, kb[0], 3)
		require.Len(t, kb[1], 7)

		last := kb[len(kb)-1]
		require.Len(t, last, 1)
		assert.Equal(t, "cancel", last[0].Data)

		// Not the current month, so both nav arrows are live.
		assert.Equal(t, "cal:2019-12", kb[0][0].Data)
		assert.Equal(t, "cal:2020-02", kb[0][2].Data)

		found := map[string]bool{}
		for _, row := range kb {
			for _, b := range row {
				found[b.Data] = true
			}
		}

		assert.True(t, found["pick:2020-01-01"])
		assert.True(t, found["pick:2020-01-31"])
	})

	t.Run("the current month disables paging forward and blanks future days", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		kb := calendar.Keyboard(now.Year(), now.Month(), cb)

		assert.Equal(t, "noop", kb[0][2].Data)

		found := map[string]bool{}
		for _, row := range kb {
			for _, b := range row {
				found[b.Data] = true
			}
		}

		tomorrow := now.AddDate(0, 0, 1)
		if tomorrow.Month() == now.Month() {
			assert.False(t, found["pick:"+tomorrow.Format("2006-01-02")])
		}

		assert.True(t, found["pick:"+now.Format("2006-01-02")])
	})

	t.Run("no cancel row when Cancel is empty", func(t *testing.T) {
		t.Parallel()

		kb := calendar.Keyboard(2020, time.January, calendar.Callbacks{
			MonthPrefix: "cal:", DayPrefix: "pick:", Noop: "noop",
		})

		for _, row := range kb {
			for _, b := range row {
				assert.NotEqual(t, "Отмена", b.Label)
			}
		}
	})
}

func TestAddMonths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		year      int
		month     time.Month
		delta     int
		wantYear  int
		wantMonth time.Month
	}{
		{name: "forward within year", year: 2026, month: time.March, delta: 1, wantYear: 2026, wantMonth: time.April},
		{name: "backward within year", year: 2026, month: time.March, delta: -1, wantYear: 2026, wantMonth: time.February},
		{name: "forward across year boundary", year: 2026, month: time.December, delta: 1, wantYear: 2027, wantMonth: time.January},
		{name: "backward across year boundary", year: 2026, month: time.January, delta: -1, wantYear: 2025, wantMonth: time.December},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			year, month := calendar.AddMonths(tc.year, tc.month, tc.delta)

			assert.Equal(t, tc.wantYear, year)
			assert.Equal(t, tc.wantMonth, month)
		})
	}
}

func TestParseYearMonth(t *testing.T) {
	t.Parallel()

	year, month, ok := calendar.ParseYearMonth("2026-03")
	require.True(t, ok)
	assert.Equal(t, 2026, year)
	assert.Equal(t, time.March, month)

	_, _, ok = calendar.ParseYearMonth("not-a-month")
	assert.False(t, ok)
}

func TestParseDate(t *testing.T) {
	t.Parallel()

	date, ok := calendar.ParseDate("2026-03-15")
	require.True(t, ok)
	assert.Equal(t, "2026-03-15", date.Format("2006-01-02"))

	_, ok = calendar.ParseDate("not-a-date")
	assert.False(t, ok)
}
