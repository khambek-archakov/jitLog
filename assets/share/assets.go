// Package shareassets holds every binary asset the monthly share image
// (see internal/share) draws from — fonts, logo, seal — each pulled in
// via go:embed so the built binary never reads them from disk at
// runtime. This package exists only because go:embed can't reach files
// outside its own directory tree; internal/share (the actual drawing
// code, kept separate from the Telegram handler per its own package doc)
// is this package's only consumer.
package shareassets

import _ "embed"

//go:embed logo.png
var LogoPNG []byte

//go:embed seal.png
var SealPNG []byte

//go:embed seigaiha_bottom.png
var SeigaihaBottomPNG []byte

//go:embed fonts/Inter-Regular.ttf
var FontRegular []byte

//go:embed fonts/Inter-SemiBold.ttf
var FontSemiBold []byte

//go:embed fonts/Inter-ExtraBold.ttf
var FontExtraBold []byte
