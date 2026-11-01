package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func calDate(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil {
		panic(err)
	}

	return t
}

// TestCalDayRows_September2026 guards the spec's own worked example: 1
// September 2026 is a Tuesday, so the grid's first week starts with
// August's own days and the last week ends in October's — always 42
// cells regardless.
func TestCalDayRows_September2026(t *testing.T) {
	t.Parallel()

	month := calDate("2026-09-01")
	rows := calDayRows(month, calFromDayCallback)

	require.Len(t, rows, 6)

	var total int
	for _, row := range rows {
		require.Len(t, row, 7)
		total += len(row)
	}

	assert.Equal(t, 42, total)

	first := rows[0][0]
	assert.Equal(t, "(31)", first.Label, "first week should start with August's own 31st")
	assert.Equal(t, calFromDayCallback(calDate("2026-08-31")), first.Data)

	last := rows[5][6]
	assert.Equal(t, "(11)", last.Label, "last week should end with October's own 11th")
	assert.Equal(t, calFromDayCallback(calDate("2026-10-11")), last.Data)

	// The open month's own 1st is plain, no parens.
	assert.Equal(t, "1", rows[0][1].Label)
	assert.Equal(t, calFromDayCallback(month), rows[0][1].Data)
}

// TestCalDayRows_February2027 guards the spec's own second worked
// example: February 2027 only spans 4 calendar weeks on its own, but the
// grid always pads out to the same 42 cells.
func TestCalDayRows_February2027(t *testing.T) {
	t.Parallel()

	rows := calDayRows(calDate("2027-02-01"), calFromDayCallback)

	require.Len(t, rows, 6)

	var total int
	for _, row := range rows {
		total += len(row)
	}

	assert.Equal(t, 42, total)
}

// TestCalDayRows_PickingAnAdjacentMonthDay guards the spec's own scenario:
// picking an August day from the September grid (1 September 2026 is a
// Tuesday, so the grid's only August cell is the 31st) must carry
// August's own full date as a valid "От", not get coerced into
// September.
func TestCalDayRows_PickingAnAdjacentMonthDay(t *testing.T) {
	t.Parallel()

	rows := calDayRows(calDate("2026-09-01"), calFromDayCallback)

	aug31 := rows[0][0]
	assert.Equal(t, "(31)", aug31.Label)
	assert.Equal(t, "cal:d:20260831", aug31.Data)

	got, ok := parseCalFromDay(aug31.Data)
	require.True(t, ok)
	assert.True(t, got.Equal(calDate("2026-08-31")))
}

func TestCalDayLabel(t *testing.T) {
	t.Parallel()

	sept := calDate("2026-09-01")

	assert.Equal(t, "15", calDayLabel(calDate("2026-09-15"), sept))
	assert.Equal(t, "(31)", calDayLabel(calDate("2026-08-31"), sept))
	assert.Equal(t, "(5)", calDayLabel(calDate("2026-10-05"), sept))
}

// TestCalDayRows_AnyDateIsPickable guards the spec's own removal of every
// old unavailability rule: future days, days before any training, and a
// weekend/weekday mix are all equally pickable now.
func TestCalDayRows_AnyDateIsPickable(t *testing.T) {
	t.Parallel()

	// Deliberately a month far in the future relative to any real "now".
	rows := calDayRows(calDate("2099-09-01"), calFromDayCallback)

	for _, row := range rows {
		for _, btn := range row {
			assert.NotEqual(t, "·", btn.Label)
			assert.NotEqual(t, callbackCalNoop, btn.Data)
		}
	}
}

func TestCalDayRows_OutsideAbsoluteRangeIsInert(t *testing.T) {
	t.Parallel()

	// January 2000's own grid necessarily reaches back into December
	// 1999, which falls outside [calAbsoluteMin, calAbsoluteMax].
	rows := calDayRows(calAbsoluteMin, calFromDayCallback)

	first := rows[0][0]
	assert.Equal(t, "·", first.Label)
	assert.Equal(t, callbackCalNoop, first.Data)
}

func TestAddCalMonths_CrossesYearBoundary(t *testing.T) {
	t.Parallel()

	dec := calDate("2026-12-15")

	jan := addCalMonths(dec, 1)
	assert.Equal(t, "2027-01-01", jan.Format("2006-01-02"))

	nov := addCalMonths(dec, -1)
	assert.Equal(t, "2026-11-01", nov.Format("2006-01-02"))

	nextYear := addCalMonths(dec, 12)
	assert.Equal(t, "2027-12-01", nextYear.Format("2006-01-02"))

	prevYear := addCalMonths(dec, -12)
	assert.Equal(t, "2025-12-01", prevYear.Format("2006-01-02"))
}

func TestCalNavRow_FourArrowsNoLabel(t *testing.T) {
	t.Parallel()

	row := calNavRow(calDate("2026-10-01"), calFromMonthCallback)

	require.Len(t, row, 4)
	assert.Equal(t, "«", row[0].Label)
	assert.Equal(t, "‹", row[1].Label)
	assert.Equal(t, "›", row[2].Label)
	assert.Equal(t, "»", row[3].Label)

	assert.Equal(t, "cal:f:202609", row[1].Data) // ‹ one month back
	assert.Equal(t, "cal:f:202611", row[2].Data) // › one month forward
}

func TestCalNavRow_ClampsAtAbsoluteBoundary(t *testing.T) {
	t.Parallel()

	t.Run("at the earliest month, back arrows are ignored but still labeled", func(t *testing.T) {
		t.Parallel()

		row := calNavRow(calAbsoluteMin, calFromMonthCallback)

		assert.Equal(t, "«", row[0].Label)
		assert.Equal(t, callbackCalNoop, row[0].Data)
		assert.Equal(t, "‹", row[1].Label)
		assert.Equal(t, callbackCalNoop, row[1].Data)

		assert.NotEqual(t, callbackCalNoop, row[2].Data) // › still live
	})

	t.Run("at the latest month, forward arrows are ignored but still labeled", func(t *testing.T) {
		t.Parallel()

		row := calNavRow(startOfMonth(calAbsoluteMax), calFromMonthCallback)

		assert.NotEqual(t, callbackCalNoop, row[1].Data) // ‹ still live

		assert.Equal(t, "›", row[2].Label)
		assert.Equal(t, callbackCalNoop, row[2].Data)
		assert.Equal(t, "»", row[3].Label)
		assert.Equal(t, callbackCalNoop, row[3].Data)
	})

	t.Run("in the middle, every arrow is live", func(t *testing.T) {
		t.Parallel()

		row := calNavRow(calDate("2050-06-01"), calFromMonthCallback)

		for _, btn := range row {
			assert.NotEqual(t, callbackCalNoop, btn.Data)
		}
	})
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

// TestCalCallbacks_FitWithin64Bytes guards the spec's own hard limit.
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
	}

	for _, cb := range callbacks {
		assert.LessOrEqual(t, len(cb), 64, "callback %q exceeds 64 bytes", cb)
	}
}

func TestCalFromText_BoldsTheOpenMonth(t *testing.T) {
	t.Parallel()

	text := calFromText(calDate("2026-10-01"))

	assert.Equal(t, "Выбери начало периода\n*Октябрь 2026*", text)
}

func TestCalToText_BoldsTheOpenMonth_FollowsFromAcrossMonths(t *testing.T) {
	t.Parallel()

	from := calDate("2026-10-15")

	// The open month during navigation can drift away from from's own
	// month — the bold line always reflects whatever's currently open.
	text := calToText(from, calDate("2026-09-01"))

	assert.Contains(t, text, "Выбери конец периода")
	assert.Contains(t, text, "Начало: 15 октября 2026")
	assert.Contains(t, text, "*Сентябрь 2026*")
}

func TestEscapeMarkdownV2(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `Октябрь 2026`, escapeMarkdownV2("Октябрь 2026"))
	assert.Equal(t, `\(28\)`, escapeMarkdownV2("(28)"))
	assert.Equal(t, `15 октября 2026\.`, escapeMarkdownV2("15 октября 2026."))
	assert.Equal(t, `a\_b\*c`, escapeMarkdownV2("a_b*c"))
}
