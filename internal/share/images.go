package share

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg" // logo.png's own bytes are actually JPEG-encoded, despite the extension.
	_ "image/png"

	shareassets "github.com/khambek-archakov/jitLog/assets/share"
)

var (
	logoImage image.Image
	sealImage image.Image
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
}
