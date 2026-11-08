package share

import (
	"testing"
	"time"

	"github.com/fogleman/gg"
	"github.com/stretchr/testify/assert"
)

// legendFooterGapFloor is the spec's own minimum clearance between the
// legend and the footer's top rule, at the final 1080×1920 output —
// halved here since this test works in the same logical space as the
// drawing code (see share.go's outputScale).
const legendFooterGapFloor = 120.0 / outputScale

// TestLegendClearsFooter guards three datasets — the densest shipped
// case (three types, a rounds block), a single-type/no-rounds case, and
// the long-legend/wrap case — against the exact regression this test was
// added for: the legend and the footer's own top rule drawing on top of
// (or too close to) each other, because the footer's own bottom-anchored
// position never accounted for how far down the legend above it actually
// reaches.
func TestLegendClearsFooter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Input
	}{
		{name: "three types, with a rounds block", in: Input{
			Name: "Алексей", Month: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			Trainings: 14, TotalMinutes: 21 * 60, Rounds: 63,
			GiCount: 8, NoGiCount: 4, OpenMatCount: 2,
		}},
		{name: "one type, no rounds block", in: Input{
			Name: "Мария", Month: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			Trainings: 1, TotalMinutes: 60, Rounds: 0,
			GiCount: 1,
		}},
		{name: "long legend, wrapped", in: Input{
			Name: "Алексей", Month: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			Trainings: 329, TotalMinutes: 329 * 60, Rounds: 1200,
			GiCount: 128, NoGiCount: 104, OpenMatCount: 97,
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dc := gg.NewContext(logicalW*outputScale, logicalH*outputScale)
			dc.Scale(outputScale, outputScale)

			legendBottom := drawTypeBar(dc, tc.in, nil)
			lineTop := targetFooterLineCapTop

			gap := lineTop - legendBottom
			assert.GreaterOrEqualf(
				t, gap, legendFooterGapFloor, "legend bottom (%.1f) to footer line top (%.1f) is only %.1f, want ≥ %.1f",
				legendBottom, lineTop, gap, legendFooterGapFloor,
			)
		})
	}
}

// TestDrawLegend_NeverOverflows guards the shrink-then-wrap ladder itself
// — regardless of how wide the three counts get, no row the legend
// actually draws may run past contentRight. There's no realistic upper
// bound on a type's count (a very active month), so this is checked at
// the pathological end, not just the shipped sample numbers.
func TestDrawLegend_NeverOverflows(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		segs []typeSegment
	}{
		{name: "the spec's own long-legend example", segs: []typeSegment{
			{"Gi", 128}, {"No-Gi", 104}, {"Open Mat", 97},
		}},
		{name: "short counts never needed the ladder in the first place", segs: []typeSegment{
			{"Gi", 8}, {"No-Gi", 4}, {"Open Mat", 2},
		}},
		{name: "one absurdly large count, alone", segs: []typeSegment{
			{"Open Mat", 999999},
		}},
		{name: "three absurdly large counts", segs: []typeSegment{
			{"Gi", 123456}, {"No-Gi", 123456}, {"Open Mat", 123456},
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dc := gg.NewContext(logicalW, logicalH)

			size, gap := legendSizeDefault, legendGapDefault

			if legendWidth(dc, tc.segs, size, gap) > contentWidth {
				gap = legendGapFloor
			}

			if legendWidth(dc, tc.segs, size, gap) > contentWidth {
				size = legendSizeFloor
			}

			rowSegs := tc.segs
			if legendWidth(dc, tc.segs, size, gap) > contentWidth && len(tc.segs) > 1 {
				rowSegs = tc.segs[:len(tc.segs)-1]

				lastRowWidth := legendWidth(dc, tc.segs[len(tc.segs)-1:], size, gap)
				assert.LessOrEqualf(
					t, lastRowWidth, contentWidth, "wrapped last item still overflows: %.1f > %.1f", lastRowWidth, contentWidth,
				)
			}

			rowWidth := legendWidth(dc, rowSegs, size, gap)
			assert.LessOrEqualf(t, rowWidth, contentWidth, "first row overflows: %.1f > %.1f", rowWidth, contentWidth)
		})
	}
}

func TestLegendWidth_ShrinksWithSmallerGapAndSize(t *testing.T) {
	t.Parallel()

	dc := gg.NewContext(logicalW, logicalH)
	segs := []typeSegment{{"Gi", 128}, {"No-Gi", 104}, {"Open Mat", 97}}

	full := legendWidth(dc, segs, legendSizeDefault, legendGapDefault)
	tighterGap := legendWidth(dc, segs, legendSizeDefault, legendGapFloor)
	tighterBoth := legendWidth(dc, segs, legendSizeFloor, legendGapFloor)

	assert.Less(t, tighterGap, full)
	assert.Less(t, tighterBoth, tighterGap)
}
