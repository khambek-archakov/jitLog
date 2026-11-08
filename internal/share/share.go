// Package share renders the "📤 Поделиться" monthly recap image — a
// self-contained drawing package with no Telegram/usecase dependencies
// at all (see internal/usecase/training/stats, the only caller, which
// aggregates one month's trainings into Input and does all the sending).
package share

import (
	"bytes"
	"fmt"
	"time"

	"github.com/fogleman/gg"
)

const (
	logicalW, logicalH = 540, 960
	outputScale        = 2

	marginX = 96.0
	// marginTop/marginBottom carve out Instagram/Telegram's own story UI
	// safe zone — their own chrome covers the top ~270px and bottom
	// ~384px of a published 1080×1920 story, hence 135/192 here (half of
	// that, this space being drawn at outputScale=2). Content must stay
	// within final y ∈ [270, 1536] — see TestContentStaysInSafeZone.
	marginTop    = 135.0
	marginBottom = 192.0
)

var (
	contentLeft  = marginX
	contentRight = float64(logicalW) - marginX
	contentWidth = contentRight - contentLeft
)

// Input is one month's worth of already-aggregated stats — the caller
// (internal/usecase/training/stats) owns every bit of business logic
// (which month, what counts as "this month's trainings"); this package
// only knows how to draw them.
type Input struct {
	Name         string
	Month        time.Time // only Year()/Month() are read.
	Trainings    int
	TotalMinutes int32
	Rounds       int32
	GiCount      int
	NoGiCount    int
	OpenMatCount int
}

// Render draws Input as a 1080×1920 PNG, entirely in memory — no file
// ever touches disk.
func Render(in Input) ([]byte, error) {
	dc := gg.NewContext(logicalW*outputScale, logicalH*outputScale)
	dc.Scale(outputScale, outputScale)

	dc.SetColor(colorBg)
	dc.Clear()

	drawDecor(dc)

	y := drawHeader(dc, in.Name)
	y = drawTitle(dc, y, in.Month)
	y = drawTrainingsCount(dc, y, in.Trainings)
	y = drawHoursAndRounds(dc, y, in.TotalMinutes, in.Rounds)
	y = drawTypeBar(dc, y, in)
	drawFooter(dc)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, fmt.Errorf("encode share image: %w", err)
	}

	return buf.Bytes(), nil
}
