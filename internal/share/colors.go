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

var (
	colorBg       = hex("#F4F1EA")
	colorText     = hex("#16181D")
	colorSub      = hex("#4A4F5A")
	colorAccent   = hex("#2B45C9")
	colorBarThird = hex("#9AA3C7")

	// typeColors keys by this package's own three training-type buckets
	// (see Input) — fixed by the design spec, not caller-configurable.
	typeColors = map[string]color.Color{
		"Gi":       colorText,
		"No-Gi":    colorAccent,
		"Open Mat": colorBarThird,
	}
)
