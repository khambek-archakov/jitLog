// Package share renders the "📤 Поделиться" month/year recap image — a
// self-contained drawing package with no Telegram/usecase dependencies
// at all (see internal/usecase/training/stats, the only caller, which
// aggregates one period's trainings into Input and does all the
// sending).
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

// Period picks which recap card is drawn. The zero value, PeriodMonth,
// keeps every existing caller's behavior unchanged: label "МОЙ МЕСЯЦ НА
// ТАТАМИ", title the month's own name + year (see monthTitle).
// PeriodYear draws label "МОЙ ГОД НА ТАТАМИ" and a digits-only year
// title, and lets the big-number blocks (trainings/hours/rounds) shrink
// to fit — see drawTrainingsCount/drawHoursAndRounds's own doc comments
// — since a year's totals routinely run into 3-4 digits where a
// month's never do. Everything else (layout, sizes, colors, positions,
// safe zone) is identical between the two.
type Period int

const (
	PeriodMonth Period = iota
	PeriodYear
)

// Input is one period's worth of already-aggregated stats — the caller
// (internal/usecase/training/stats) owns every bit of business logic
// (which period, what counts as "this period's trainings"); this
// package only knows how to draw them.
type Input struct {
	Period       Period
	Name         string
	Month        time.Time // only Year() is read for PeriodYear; Year()/Month() for PeriodMonth.
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

	dc.SetColor(colorBg)
	dc.Clear()

	// Drawn in raw final-pixel space, before dc.Scale below switches
	// everything else (including drawDecor, right after) to the logical
	// (540×960) coordinate system this package otherwise draws in.
	// seigaihaBottomImage is already sized for the final 1080×1920
	// canvas (1080×440), bottom-aligned at y=1480 (1480+440=1920), so it
	// needs no scaling of its own — see drawSeigaihaBottom's own doc
	// comment on why this one image alone draws pre-scale.
	drawSeigaihaBottom(dc)

	dc.Scale(outputScale, outputScale)

	drawDecor(dc)

	drawHeader(dc, in.Name, tr)
	drawTitle(dc, in.Period, in.Month, tr)
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
