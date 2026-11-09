package share

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// yearTestInputs are the three datasets the "МОЙ ГОД НА ТАТАМИ" card's
// own spec named: a dense year (three types, four-digit rounds), a
// sparse one (a single type, no rounds block), and a long name paired
// with very large numbers (365/999/4800) — the same three cases
// safezone_test.go's TestContentStaysInSafeZone and layout_test.go's
// TestLegendClearsFooter also render.
var yearTestInputs = []Input{
	{
		Period: PeriodYear,
		Name:   "Алексей", Month: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Trainings: 187, TotalMinutes: 312 * 60, Rounds: 1204,
		GiCount: 110, NoGiCount: 60, OpenMatCount: 17,
	},
	{
		Period: PeriodYear,
		Name:   "Алексей", Month: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Trainings: 4, TotalMinutes: 6 * 60, Rounds: 0,
		NoGiCount: 4,
	},
	{
		Period:    PeriodYear,
		Name:      "Константин Александрович Долгопрудненский-Воронцов",
		Month:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Trainings: 365, TotalMinutes: 999 * 60, Rounds: 4800,
		GiCount: 200, NoGiCount: 100, OpenMatCount: 65,
	},
}

// TestYearCardBigNumbersDontOverlap renders each of yearTestInputs and
// checks every named block's own cap-top still lands within the same
// ±8px tolerance of the month card's own targets (the year card shares
// every position with the month card, per the spec — only the label,
// title and big-number shrink logic differ), and that no block overlaps
// the one before it in the vertical flow — reusing wantedLayout and
// checkNoOverlaps from reference_test.go, which already do exactly this
// for the month card's own two reference datasets.
func TestYearCardBigNumbersDontOverlap(t *testing.T) {
	t.Parallel()

	for i, in := range yearTestInputs {
		t.Run(in.Name, func(t *testing.T) {
			t.Parallel()

			_, tr, err := renderTraced(in, &trace{})
			require.NoError(t, err)

			byName := make(map[string]blockPosition, len(tr.blocks))
			for _, b := range tr.blocks {
				byName[b.name] = b
			}

			for _, w := range wantedLayout {
				b, ok := byName[w.name]
				if !ok {
					continue // "раунды" is absent from the no-rounds dataset.
				}

				d := b.capTop - w.capTop
				if d < -w.tolerance || d > w.tolerance {
					t.Errorf("dataset %d: block %q: cap-top got %.1f, want %.1f ±%.1f (off by %.1f)",
						i, w.name, b.capTop, w.capTop, w.tolerance, d)
				}
			}

			checkNoOverlaps(t, byName)
		})
	}
}
