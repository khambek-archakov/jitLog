package share

import (
	"testing"

	"github.com/fogleman/gg"
	"github.com/stretchr/testify/assert"
)

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
