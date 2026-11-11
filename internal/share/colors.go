package share

import (
	"fmt"
	"image/color"
	"strconv"
)

// hex parses a "#RRGGBB" literal — panics on a malformed one, since every
// call site here is a compile-time-known constant, never user input.
func hex(s string) color.Color {
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		panic(fmt.Errorf("parse color %q: %w", s, err))
	}

	return color.RGBA{R: uint8(v >> 16), G: uint8(v >> 8), B: uint8(v), A: 255}
}

// colorBg/colorText/colorSub are the card's entire palette now, besides
// the three per-type colors in typeColors below — a deliberate
// simplification (one accent: the trainings-count number; everything
// else is either colorText or colorSub). The earlier blue/lavender
// accent colors are gone for good, not just unused: #2B45C9 and
// #9AA3C7 appear nowhere else in this package. seal.png's own red is
// the one color this simplification doesn't touch — it's baked into
// that image, not drawn by any code here.
var (
	colorBg   = hex("#F4F1EA")
	colorText = hex("#16181D")
	colorSub  = hex("#4A4F5A")

	// typeColors keys by this package's own three training-type buckets
	// (see Input) — fixed by the design spec, not caller-configurable.
	typeColors = map[string]color.Color{
		"Gi":       colorText,
		"No-Gi":    hex("#7A808C"),
		"Open Mat": hex("#C4C7CE"),
	}
)
