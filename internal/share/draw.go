package share

import (
	"image"

	"github.com/fogleman/gg"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
)

// lineMetrics reads a face's own metrics in logical pixels — DPI 72 (see
// fonts.go's face()) makes a fixed.Int26_6 unit exactly 1/64th of a
// pixel, hence the /64.0. height is the font's own recommended line box
// (Metrics().Height — the same value gg.Context.SetFontFace itself uses
// as dc.fontHeight); ascent is scaled down proportionally from the raw
// Ascent/Descent split so a baseline placed at top+ascent never pushes
// the glyphs' own descenders past top+height. Inter's raw Ascent+Descent
// comfortably exceeds its own Height (room for scripts far outside
// Cyrillic/Latin) — using the raw Ascent directly as a baseline offset
// was overflowing every section into the one below it.
func lineMetrics(f font.Face) (ascent, height float64) {
	m := f.Metrics()
	height = float64(m.Height) / 64.0

	rawAscent := float64(m.Ascent) / 64.0
	rawDescent := float64(m.Descent) / 64.0

	if rawAscent+rawDescent <= 0 {
		return height * 0.8, height
	}

	return rawAscent * height / (rawAscent + rawDescent), height
}

// scaleToHeight resizes img to height h, preserving its own aspect
// ratio — used for logo.png, whose natural size never matches the
// layout's own 128px header height.
func scaleToHeight(img image.Image, h int) image.Image {
	b := img.Bounds()
	w := int(float64(b.Dx()) * float64(h) / float64(b.Dy()))

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)

	return dst
}

// inkBounds is s's own true ink extent in face, relative to its own
// baseline (negative = above baseline) — via font.BoundString, not the
// font's generic ascent/descent metrics, which (per this feature's own
// debug test) sit well outside where any specific string's ink actually
// falls.
func inkBounds(w weight, size float64, s string) (minY, maxY float64) {
	bounds, _ := font.BoundString(face(w, size), s)

	return float64(bounds.Min.Y) / 64, float64(bounds.Max.Y) / 64
}

// baselineForCapTop returns the baseline y that puts s's own true ink
// top exactly at capTop, for a string drawn in face w at size — the
// layout's own primary positioning primitive: every text block is placed
// by its target cap-top, not by accumulating gaps/ascents downward (the
// approach TestDebugLayoutPositions found drifted 60-190px off target).
func baselineForCapTop(w weight, size, capTop float64, s string) float64 {
	minY, _ := inkBounds(w, size, s)

	return capTop - minY
}

// drawTracked draws s letter by letter with spacing extra pixels (can be
// negative, for slightly tighter-than-normal tracking) between glyphs,
// baseline-left anchored at (x, y). Returns the total width drawn.
func drawTracked(dc *gg.Context, s string, x, y, spacing float64) float64 {
	cx := x

	for _, r := range s {
		ch := string(r)
		dc.DrawString(ch, cx, y)

		w, _ := dc.MeasureString(ch)
		cx += w + spacing
	}

	return cx - x
}

// measureTracked is drawTracked's own width, without drawing anything —
// kept in sync with it deliberately (same per-glyph advance math) rather
// than factored into one function, since drawTracked needs to draw as it
// measures and this doesn't.
func measureTracked(dc *gg.Context, s string, spacing float64) float64 {
	var w float64

	for _, r := range s {
		cw, _ := dc.MeasureString(string(r))
		w += cw + spacing
	}

	return w
}

// measureString is s's own advance width in face w at size, without a
// gg.Context — used by fitNumberSize, which needs to probe several
// candidate sizes before a context's own font face is set to any of
// them.
func measureString(w weight, size float64, s string) float64 {
	return float64(font.MeasureString(face(w, size), s)) / 64
}

// fitNumberSize returns the largest size, stepping down from base in
// shrinkStep increments, at which s measures no wider than availWidth —
// or shrinkFloorRatio*base itself once that's reached, even if s still
// doesn't fit at that floor (the year card's own "big number" blocks
// shrink to fit, per the spec, but never truncate the number itself).
func fitNumberSize(w weight, base, availWidth float64, s string) float64 {
	const (
		shrinkStep       = 4.0 // 8 final.
		shrinkFloorRatio = 0.7
	)

	floor := base * shrinkFloorRatio

	size := base
	for measureString(w, size, s) > availWidth && size > floor {
		size -= shrinkStep
		if size < floor {
			size = floor
		}
	}

	return size
}
