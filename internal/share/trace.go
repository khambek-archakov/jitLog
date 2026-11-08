package share

import "golang.org/x/image/font"

// blockPosition is one named block's own measured position, in final
// (1080×1920) pixels. capTop is the glyph's true ink top — via
// font.BoundString, not the font face's generic ascent box, which
// (as TestDebugLayoutPositions found) sits well above where the ink
// actually starts — baseline is 0 for blocks with no baseline concept
// (images, the bar).
type blockPosition struct {
	name     string
	capTop   float64
	baseline float64
}

// trace collects blockPositions during a render. A nil *trace (every
// production Render() call — see share.go) makes every method here a
// no-op, so tracing costs nothing outside tests.
type trace struct {
	blocks []blockPosition
}

// image records an image block's own top edge (its "cap-top" — there's
// no baseline for an image).
func (t *trace) image(name string, topLogical float64) {
	if t == nil {
		return
	}

	t.blocks = append(t.blocks, blockPosition{name: name, capTop: topLogical * outputScale})
}

// shape records a non-text, non-image block's own top edge (e.g. the
// type bar, the footer's rule line).
func (t *trace) shape(name string, topLogical float64) {
	if t == nil {
		return
	}

	t.blocks = append(t.blocks, blockPosition{name: name, capTop: topLogical * outputScale})
}

// text records a text block's true cap-top (measured via
// font.BoundString against the exact string drawn, not assumed from the
// face's own ascent metric) and baseline.
func (t *trace) text(name string, w weight, size, baselineLogical float64, s string) {
	if t == nil {
		return
	}

	bounds, _ := font.BoundString(face(w, size), s)
	capTopLogical := baselineLogical + float64(bounds.Min.Y)/64

	t.blocks = append(t.blocks, blockPosition{
		name:     name,
		capTop:   capTopLogical * outputScale,
		baseline: baselineLogical * outputScale,
	})
}
