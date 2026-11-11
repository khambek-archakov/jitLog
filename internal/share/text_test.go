package share

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMonthTitle(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Октябрь 2026", monthTitle(time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)))
	assert.Equal(t, "Январь 2027", monthTitle(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)))
	assert.Equal(t, "Декабрь 2025", monthTitle(time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)))
}

func TestPeriodTitle(t *testing.T) {
	t.Parallel()

	label, title := periodTitle(PeriodMonth, time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, "Мой месяц на татами", label)
	assert.Equal(t, "Октябрь 2026", title)

	label, title = periodTitle(PeriodYear, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	assert.Equal(t, "Мой год на татами", label)
	assert.Equal(t, "2026", title)
}

func TestFormatThousands(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "0", formatThousands(0))
	assert.Equal(t, "187", formatThousands(187))
	assert.Equal(t, "999", formatThousands(999))
	assert.Equal(t, "1 204", formatThousands(1204))
	assert.Equal(t, "4 800", formatThousands(4800))
	assert.Equal(t, "10 000", formatThousands(10000))
}

func TestRoundHours(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		minutes int32
		want    int
	}{
		{name: "zero stays zero", minutes: 0, want: 0},
		{name: "a short session still shows 1, never 0", minutes: 15, want: 1},
		{name: "29 minutes still rounds up to 1", minutes: 29, want: 1},
		{name: "exactly one hour", minutes: 60, want: 1},
		{name: "rounds down", minutes: 89, want: 1},
		{name: "rounds up", minutes: 91, want: 2},
		{name: "21 hours exactly", minutes: 21 * 60, want: 21},
		{name: "99 hours exactly", minutes: 99 * 60, want: 99},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, roundHours(tc.minutes))
		})
	}
}

func TestPluralWords(t *testing.T) {
	t.Parallel()

	t.Run("trainingsWord", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "тренировка", trainingsWord(1))
		assert.Equal(t, "тренировка", trainingsWord(21))
		assert.Equal(t, "тренировка", trainingsWord(31))
		assert.Equal(t, "тренировки", trainingsWord(2))
		assert.Equal(t, "тренировки", trainingsWord(3))
		assert.Equal(t, "тренировки", trainingsWord(4))
		assert.Equal(t, "тренировки", trainingsWord(22))
		assert.Equal(t, "тренировок", trainingsWord(0))
		assert.Equal(t, "тренировок", trainingsWord(5))
		assert.Equal(t, "тренировок", trainingsWord(11))
		assert.Equal(t, "тренировок", trainingsWord(14))
		assert.Equal(t, "тренировок", trainingsWord(14 /* the spec's own worked example */))
	})

	t.Run("hoursWord", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "час", hoursWord(1))
		assert.Equal(t, "часа", hoursWord(2))
		assert.Equal(t, "часа", hoursWord(3))
		assert.Equal(t, "часа", hoursWord(4))
		assert.Equal(t, "часов", hoursWord(0))
		assert.Equal(t, "часов", hoursWord(11))
		assert.Equal(t, "час", hoursWord(21)) // 21%10==1 and 21%100 isn't 11-14.
		assert.Equal(t, "часов", hoursWord(99))
	})

	// The year card's own big numbers can run well past 100 (e.g. 187
	// trainings) — this covers the spec's own worked set, including the
	// two three-digit cases (111 many, 121 one) a month's numbers never
	// reach.
	t.Run("the spec's own worked set: 1, 2, 5, 11, 21, 22, 111, 121", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			n    int
			want string
		}{
			{1, "тренировка"}, {2, "тренировки"}, {5, "тренировок"}, {11, "тренировок"},
			{21, "тренировка"}, {22, "тренировки"}, {111, "тренировок"}, {121, "тренировка"},
		}

		for _, tc := range tests {
			assert.Equalf(t, tc.want, trainingsWord(tc.n), "trainingsWord(%d)", tc.n)
		}
	})

	t.Run("roundsWord", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "раунд", roundsWord(1))
		assert.Equal(t, "раунда", roundsWord(3))
		assert.Equal(t, "раунда", roundsWord(63)) // 63%10==3 and 63%100 isn't 11-14.
		assert.Equal(t, "раундов", roundsWord(480))
	})
}

func TestTruncateToWidth(t *testing.T) {
	t.Parallel()

	// A fake measure: every character is exactly 10 units wide, so width
	// math is exact and the test needs no real font.
	measure := func(s string) float64 {
		return float64(len([]rune(s))) * 10
	}

	t.Run("short enough — unchanged", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "Алексей", truncateToWidth("Алексей", 1000, measure))
	})

	t.Run("exactly at the limit — unchanged", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "12345", truncateToWidth("12345", 50, measure))
	})

	t.Run("too long — trimmed with an ellipsis", func(t *testing.T) {
		t.Parallel()

		got := truncateToWidth("Константин Александрович", 100, measure)

		assert.True(t, strings.HasSuffix(got, "…"))
		assert.LessOrEqual(t, measure(got), 100.0)
		assert.Less(t, len(got), len("Константин Александрович"))
	})

	t.Run("even a single character doesn't fit — just the ellipsis", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "…", truncateToWidth("Константин", 5, measure))
	})

	t.Run("a cut landing right after a word break drops the trailing space", func(t *testing.T) {
		t.Parallel()

		// "Константин " is 11 runes (110 units) — a maxWidth of 120
		// lands the cut exactly there, where the untrimmed candidate
		// would be "Константин …" (12 runes, 120 units — it also still
		// fits, which is exactly the bug: a worse candidate was
		// accepted before the better, space-trimmed one got a chance).
		got := truncateToWidth("Константин Александрович", 120, measure)

		assert.Equal(t, "Константин…", got)
		assert.False(t, strings.Contains(got, " …"))
	})
}
