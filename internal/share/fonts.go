package share

import (
	"fmt"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"

	shareassets "github.com/khambek-archakov/jitLog/assets/share"
)

// weight is one of the three static Inter cuts this package embeds —
// gg/freetype can't work with a variable font, hence static .ttf files
// (see assets/share/fonts and its own OFL.txt).
type weight int

const (
	weightRegular weight = iota
	weightSemiBold
	weightExtraBold
)

var fonts = map[weight]*truetype.Font{}

func init() {
	parse := func(data []byte, name string) *truetype.Font {
		f, err := truetype.Parse(data)
		if err != nil {
			panic(fmt.Errorf("parse embedded font %s: %w", name, err))
		}

		return f
	}

	fonts[weightRegular] = parse(shareassets.FontRegular, "Inter-Regular.ttf")
	fonts[weightSemiBold] = parse(shareassets.FontSemiBold, "Inter-SemiBold.ttf")
	fonts[weightExtraBold] = parse(shareassets.FontExtraBold, "Inter-ExtraBold.ttf")
}

// face builds a font.Face at points — DPI 72 so a "point" here is
// exactly one logical pixel in the 540×960 layout (see layout.go), the
// same convention gg.Context.LoadFontFace itself uses internally.
func face(w weight, points float64) font.Face {
	return truetype.NewFace(fonts[w], &truetype.Options{
		Size:    points,
		DPI:     72,
		Hinting: font.HintingFull,
	})
}
