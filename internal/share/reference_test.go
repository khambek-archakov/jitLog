package share

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// referenceInputs mirrors the two datasets assets/share/reference's own
// PNGs were rendered from — reference_full.png (14 trainings/21h/63
// rounds, Gi 8/No-Gi 4/Open Mat 2) and reference_single.png (1
// training/1h, no rounds, a single No-Gi session, name "Алексей") — see
// the user's own message naming both and reference_single.png's own
// legend swatch (a single blue "No-Gi 1" row), confirmed by inspecting
// the file directly.
var referenceInputs = map[string]Input{
	// reference_full.png's own title reads "Октябрь 2026",
	// reference_single.png's own reads "Ноябрь 2026" — the two reference
	// renders don't share a month, confirmed by inspecting both files
	// directly.
	"full": {
		Name: "Алексей", Month: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Trainings: 14, TotalMinutes: 21 * 60, Rounds: 63,
		GiCount: 8, NoGiCount: 4, OpenMatCount: 2,
	},
	// reference_single.png's own numbers are 14 trainings / 21 hours —
	// same as "full" — it's "single" in the sense of a single legend
	// item (No-Gi 1) and no rounds block, not a single training;
	// confirmed by inspecting the file directly (it does not match an
	// earlier, mistaken description of "1 training/1h").
	"single": {
		Name: "Алексей", Month: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC),
		Trainings: 14, TotalMinutes: 21 * 60, Rounds: 0,
		NoGiCount: 1,
	},
}

// blockHeight is each non-text block's own known height, in final px —
// used by TestMatchesReference's overlap check, since only text blocks
// carry a baseline (a text block's own baseline stands in for its
// bottom there instead; see that check's own comment).
var blockHeight = map[string]float64{
	"логотип":      112,
	"полоса":       28,
	"линия футера": 4,
}

// TestMatchesReference renders both of assets/share/reference's own
// datasets, diffs each render against its reference PNG pixel-by-pixel
// (saving the diff image alongside, in the OS temp dir, for visual
// inspection), and checks every named block against
// reference_layout.json's own box tops (reusing wantedLayout — the same
// table TestDebugLayoutPositions checks, since every target is
// data-independent) and against its own immediate predecessor in the
// vertical flow, for a hard overlap check.
func TestMatchesReference(t *testing.T) {
	for _, name := range []string{"full", "single"} {
		t.Run(name, func(t *testing.T) {
			in := referenceInputs[name]

			data, tr, err := renderTraced(in, &trace{})
			if err != nil {
				t.Fatalf("render: %v", err)
			}

			got, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatalf("decode own render: %v", err)
			}

			refPath := filepath.Join("..", "..", "assets", "share", "reference", "reference_"+name+".png")

			refFile, err := os.Open(refPath)
			if err != nil {
				t.Fatalf("open %s: %v", refPath, err)
			}
			defer refFile.Close()

			want, err := png.Decode(refFile)
			if err != nil {
				t.Fatalf("decode %s: %v", refPath, err)
			}

			if got.Bounds() != want.Bounds() {
				t.Fatalf("own render is %v, reference is %v", got.Bounds(), want.Bounds())
			}

			diffPath := filepath.Join(os.TempDir(), "share_diff_"+name+".png")
			pct := saveDiff(t, got, want, diffPath)
			t.Logf("%s: %.2f%% of pixels differ by more than the threshold; diff image: %s", name, pct, diffPath)

			byName := make(map[string]blockPosition, len(tr.blocks))
			for _, b := range tr.blocks {
				byName[b.name] = b
			}

			for _, w := range wantedLayout {
				b, ok := byName[w.name]
				if !ok {
					continue // not every block exists in every dataset (e.g. "раунды" is absent from "single").
				}

				if d := b.capTop - w.capTop; d < -w.tolerance || d > w.tolerance {
					t.Errorf("block %q: cap-top got %.1f, want %.1f ±%.1f (off by %.1f)",
						w.name, b.capTop, w.capTop, w.tolerance, d)
				}
			}

			checkNoOverlaps(t, byName)
		})
	}
}

// checkNoOverlaps walks the primary vertical flow (the same block order
// wantedLayout lists, draw order top to bottom) and asserts each
// block's own bottom sits at or above the next block's own cap-top. A
// text block's baseline stands in for its bottom (descenders in this
// font are shallow enough that this never falsely reports an overlap
// in practice); image/shape blocks use blockHeight instead, since they
// carry no baseline at all.
func checkNoOverlaps(t *testing.T, byName map[string]blockPosition) {
	t.Helper()

	order := make([]string, 0, len(wantedLayout))
	for _, w := range wantedLayout {
		order = append(order, w.name)
	}

	var prev *blockPosition

	var prevName string

	for _, name := range order {
		b, ok := byName[name]
		if !ok {
			continue
		}

		if prev != nil {
			prevBottom := prev.baseline
			if h, ok := blockHeight[prevName]; ok {
				prevBottom = prev.capTop + h
			}

			if prevBottom > b.capTop {
				t.Errorf("block %q overlaps %q: %q's own bottom (%.1f) sits below %q's own cap-top (%.1f)",
					prevName, name, prevName, prevBottom, name, b.capTop)
			}
		}

		bCopy := b
		prev = &bCopy
		prevName = name
	}
}

// saveDiff writes an amplified per-pixel absolute-difference image to
// path (white = identical, darker = more different) and returns the
// percentage of pixels whose difference exceeds a visible threshold —
// purely informational (see TestMatchesReference's own doc comment on
// why this isn't asserted: two different rendering engines never agree
// pixel-for-pixel at glyph edges, so the real pass/fail signal is the
// block-position check instead).
func saveDiff(t *testing.T, got, want image.Image, path string) float64 {
	t.Helper()

	b := got.Bounds()
	diff := image.NewRGBA(b)

	var differing int

	total := b.Dx() * b.Dy()

	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			gr, gg, gb, _ := got.At(x, y).RGBA()
			wr, wg, wb, _ := want.At(x, y).RGBA()

			d := absInt(int(gr)-int(wr)) + absInt(int(gg)-int(wg)) + absInt(int(gb)-int(wb))
			if d > 3*0x2000 {
				differing++
			}

			v := uint8(255 - clampInt(d/768, 0, 255))
			diff.Set(x, y, color.RGBA{R: 255, G: v, B: v, A: 255})
		}
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	defer f.Close()

	if err := png.Encode(f, diff); err != nil {
		t.Fatalf("encode diff: %v", err)
	}

	return 100 * float64(differing) / float64(total)
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}

	return v
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}
