package share

import (
	"testing"
	"time"
)

// target is one block's expected position at the final 1080×1920 output
// — capTop always checked; baseline checked only when > 0 (the spec
// didn't give one for every block). knownDeviation, when set, turns a
// baseline miss into a logged note instead of a failure — used for
// exactly one block (see wantedLayout's own comment on "заголовок").
type target struct {
	name           string
	capTop         float64
	baseline       float64
	tolerance      float64
	knownDeviation string
}

// wantedLayout is assets/share/reference/reference_layout.json's own
// box-top table (targets.go holds the same numbers, halved into logical
// space, as the actual drawing constants) — capTop in final px, ±8px
// tolerance throughout. No baseline targets here any more: the
// reference gives CSS line-box tops, not baselines, for every block —
// see targets.go's own comment on why cap-top is the right thing to
// check against a box-top target in the first place. There's no
// "таглайн" entry any more — the tagline itself is gone (see
// drawFooter); "@JitLogBot" is the last block now.
var wantedLayout = []target{
	{name: "логотип", capTop: targetLogoCapTop * outputScale, tolerance: 8},
	{name: "надпись", capTop: targetLabelCapTop * outputScale, tolerance: 8},
	{name: "заголовок", capTop: targetTitleCapTop * outputScale, tolerance: 8},
	{name: "число тренировок", capTop: targetTrainingsCapTop * outputScale, tolerance: 8},
	{name: "часы (число)", capTop: targetHoursCapTop * outputScale, tolerance: 8},
	{name: "часы/раунды (подпись)", capTop: targetHoursLabelCapTop * outputScale, tolerance: 8},
	{name: "полоса", capTop: targetBarCapTop * outputScale, tolerance: 8},
	{name: "легенда", capTop: targetLegendCapTop * outputScale, tolerance: 8},
	{name: "линия футера", capTop: targetFooterLineCapTop * outputScale, tolerance: 8},
	{name: "@JitLogBot", capTop: targetHandleCapTop * outputScale, tolerance: 8},
}

// TestDebugLayoutPositions prints every block's own measured cap-top and
// baseline (see trace.go — cap-top is the glyph's true ink top, via
// font.BoundString, not the font face's generic ascent box) against the
// spec's own target table, for the densest shipped dataset. Run with
// `go test ./internal/share/ -run TestDebugLayoutPositions -v` to see
// the full printout regardless of pass/fail.
func TestDebugLayoutPositions(t *testing.T) {
	in := Input{
		Name: "Алексей", Month: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Trainings: 14, TotalMinutes: 21 * 60, Rounds: 63,
		GiCount: 8, NoGiCount: 4, OpenMatCount: 2,
	}

	_, tr, err := renderTraced(in, &trace{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	byName := make(map[string]blockPosition, len(tr.blocks))
	for _, b := range tr.blocks {
		byName[b.name] = b
	}

	t.Logf("%-24s %10s %10s", "block", "cap-top", "baseline")

	for _, b := range tr.blocks {
		t.Logf("%-24s %10.1f %10.1f", b.name, b.capTop, b.baseline)
	}

	for _, want := range wantedLayout {
		got, ok := byName[want.name]
		if !ok {
			t.Errorf("block %q: no trace entry recorded at all", want.name)

			continue
		}

		if want.capTop > 0 {
			if d := got.capTop - want.capTop; d < -want.tolerance || d > want.tolerance {
				t.Errorf(
					"block %q: cap-top got %.1f, want %.1f ±%.1f (off by %.1f)",
					want.name, got.capTop, want.capTop, want.tolerance, d,
				)
			}
		}

		if want.baseline > 0 {
			if d := got.baseline - want.baseline; d < -want.tolerance || d > want.tolerance {
				if want.knownDeviation != "" {
					t.Logf("block %q: baseline got %.1f, want %.1f ±%.1f — known deviation: %s",
						want.name, got.baseline, want.baseline, want.tolerance, want.knownDeviation,
					)
				} else {
					t.Errorf(
						"block %q: baseline got %.1f, want %.1f ±%.1f (off by %.1f)",
						want.name, got.baseline, want.baseline, want.tolerance, d,
					)
				}
			}
		}
	}
}
