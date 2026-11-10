package share

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // kept for robustness — not needed by any asset currently embedded, all genuine PNGs.
	_ "image/png"

	shareassets "github.com/khambek-archakov/jitLog/assets/share"
)

var (
	logoImage           image.Image
	sealImage           image.Image
	seigaihaBottomImage image.Image
)

func init() {
	decode := func(data []byte, name string) image.Image {
		img, _, err := image.Decode(bytes.NewReader(data))
		if err != nil {
			panic(fmt.Errorf("decode embedded image %s: %w", name, err))
		}

		return img
	}

	logoImage = decode(shareassets.LogoPNG, "logo.png")
	sealImage = decode(shareassets.SealPNG, "seal.png")
	seigaihaBottomImage = decode(shareassets.SeigaihaBottomPNG, "seigaiha_bottom.png")
}
