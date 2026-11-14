package share

import (
	"math"
	"strconv"
	"strings"
	"time"
)

var monthsNominative = [...]string{
	"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
	"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
}

// monthTitle is month's own nominative name plus its year, e.g. "Октябрь
// 2026" — only Year()/Month() are read, so any date within the target
// month works.
func monthTitle(month time.Time) string {
	return monthsNominative[month.Month()-1] + " " + strconv.Itoa(month.Year())
}

// periodTitle is drawTitle's own label + headline text for p/anchor,
// split out purely so the text choice is unit-testable without a real
// font/context — PeriodMonth is "Мой месяц на татами" + the month name,
// PeriodYear is "Мой год на татами" + the year, digits only. Sentence
// case, no letter-spacing — an earlier, all-caps/tracked-out version of
// this label was dropped as part of simplifying the card's visual
// language down to one accent (the trainings-count number).
func periodTitle(p Period, anchor time.Time) (label, title string) {
	if p == PeriodYear {
		return "Мой год на татами", strconv.Itoa(anchor.Year())
	}

	return "Мой месяц на татами", monthTitle(anchor)
}

// roundHours sums totalMinutes into whole hours, rounded to the nearest
// — except any positive total under 60 minutes still shows as 1, never
// 0, so a single short session doesn't vanish from its own headline
// number.
func roundHours(totalMinutes int32) int {
	if totalMinutes <= 0 {
		return 0
	}

	h := int(math.Round(float64(totalMinutes) / 60))
	if h == 0 {
		h = 1
	}

	return h
}

// pluralForm picks among three Russian plural forms by n — the standard
// one/few/many rule (see internal/usecase/training/stats.weeksWord for
// the same logic over a different word set).
func pluralForm(n int, one, few, many string) string {
	if n%100 >= 11 && n%100 <= 14 {
		return many
	}

	switch n % 10 {
	case 1:
		return one
	case 2, 3, 4:
		return few
	default:
		return many
	}
}

func trainingsWord(n int) string {
	return pluralForm(n, "тренировка", "тренировки", "тренировок")
}

func hoursWord(n int) string {
	return pluralForm(n, "час", "часа", "часов")
}

func roundsWord(n int) string {
	return pluralForm(n, "схватка", "схватки", "схваток")
}

// formatThousands renders n with a space as the thousands separator
// (e.g. 1204 -> "1 204") — a year's totals routinely run into four
// digits where a month's never do. n is never negative in this domain
// (training/hour/round counts), so that case isn't handled.
func formatThousands(n int) string {
	s := strconv.Itoa(n)

	var out []byte

	for i := range len(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ' ')
		}

		out = append(out, s[i])
	}

	return string(out)
}

// truncateToWidth returns s unchanged if measure(s) already fits within
// maxWidth, otherwise trims characters from the end and appends "…"
// until it does — trimming any trailing space off what's left first, so
// a cut that lands right after a word break never leaves a dangling
// space before the ellipsis (e.g. "Константин Александрович " + "…"
// becomes "Константин Александрович…", not "...ич …"). measure is
// injected so this stays unit-testable without a real font face — the
// drawing code passes dc.MeasureString's own width.
func truncateToWidth(s string, maxWidth float64, measure func(string) float64) string {
	if measure(s) <= maxWidth {
		return s
	}

	runes := []rune(s)

	for i := len(runes) - 1; i > 0; i-- {
		candidate := strings.TrimRight(string(runes[:i]), " ") + "…"
		if measure(candidate) <= maxWidth {
			return candidate
		}
	}

	return "…"
}
