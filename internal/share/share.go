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

	// marginX was 96 — the reference layout's own content_x is
	// [96, 984] at final (1080×1920) scale, i.e. 48 here (half that,
	// this space being drawn at outputScale=2). The old 96 doubled the
	// side margins, narrowing every block (and the name's own available
	// width, and the bar/legend) well past what the reference shows.
	marginX = 48.0
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
	data, _, err := renderTraced(in, nil)
	return data, err
}

// renderTraced is Render's own real body, plus an optional *trace: when
// non-nil, every block's measured position is recorded into it as it's
// drawn — see TestDebugLayoutPositions, the only caller that passes one.
// A nil trace (every production Render call) makes the recording calls
// below free no-ops.
func renderTraced(in Input, tr *trace) ([]byte, *trace, error) {
	dc := gg.NewContext(logicalW*outputScale, logicalH*outputScale)
	dc.Scale(outputScale, outputScale)

	dc.SetColor(colorBg)
	dc.Clear()

	drawDecor(dc)

	drawHeader(dc, in.Name, tr)
	drawTitle(dc, in.Month, tr)
	drawTrainingsCount(dc, in.Trainings, tr)
	drawHoursAndRounds(dc, in.TotalMinutes, in.Rounds, tr)
	_ = drawTypeBar(dc, in, tr)
	drawFooter(dc, tr)

	var buf bytes.Buffer
	if err := dc.EncodePNG(&buf); err != nil {
		return nil, tr, fmt.Errorf("encode share image: %w", err)
	}

	return buf.Bytes(), tr, nil
}
