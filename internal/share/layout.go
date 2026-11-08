package share

import (
	"math"
	"strconv"
	"time"

	"github.com/fogleman/gg"
)

// drawDecor paints the background's own faint "enso" arc — behind every
// other element, drawn right after the flat background fill and before
// anything else. Its center sits off-canvas to the right on purpose
// (contentRight tops out at 444); only the arc's left-hand sliver ever
// shows.
func drawDecor(dc *gg.Context) {
	dc.Push()
	defer dc.Pop()

	// The spec's own (856, 500) center and 300 radius are sized for the
	// final 1080×1920 output, not this logical 540×960 space — halved
	// here so the arc still "partially goes past the right edge" instead
	// of landing entirely off-canvas.
	const (
		cx, cy   = 428.0, 250.0
		radius   = 150.0
		width    = 26.0
		startDeg = -60.0
		sweepDeg = 340.0
	)

	dc.SetLineWidth(width)
	dc.SetLineCapRound()
	dc.SetRGBA255(0x16, 0x18, 0x1D, int(math.Round(255*0.06)))

	a1 := startDeg * math.Pi / 180
	a2 := (startDeg + sweepDeg) * math.Pi / 180

	dc.DrawArc(cx, cy, radius, a1, a2)
	dc.Stroke()
}

// drawHeader draws the logo + name row and returns the y just below it.
func drawHeader(dc *gg.Context, name string) float64 {
	const (
		logoHeight = 128.0
		logoShiftX = -8.0
		nameGap    = 20.0
		nameSize   = 30.0
	)

	logo := scaleToHeight(logoImage, int(logoHeight))
	logoW := float64(logo.Bounds().Dx())

	logoX := contentLeft + logoShiftX
	logoY := marginTop

	dc.DrawImage(logo, int(math.Round(logoX)), int(math.Round(logoY)))

	nameFace := face(weightSemiBold, nameSize)
	dc.SetFontFace(nameFace)
	dc.SetColor(colorSub)

	maxNameWidth := contentRight - (logoX + logoW) - nameGap
	name = truncateToWidth(name, maxNameWidth, func(s string) float64 {
		w, _ := dc.MeasureString(s)
		return w
	})

	nameCenterY := logoY + logoHeight/2
	dc.DrawStringAnchored(name, contentRight, nameCenterY, 1, 0.5)

	return logoY + logoHeight
}

// drawTitle draws "МОЙ МЕСЯЦ НА ТАТАМИ" + the month/year headline + the
// seal, returning the y just below the headline.
func drawTitle(dc *gg.Context, top float64, month time.Time) float64 {
	const (
		gapAfterHeader = 32.0
		labelSize      = 24.0
		labelSpacing   = 3.0
		gapLabelTitle  = 8.0
		titleSize      = 36.0
		titleSpacing   = -1.0
		sealSize       = 36.0
	)

	y := top + gapAfterHeader

	labelFace := face(weightSemiBold, labelSize)
	dc.SetFontFace(labelFace)
	dc.SetColor(colorAccent)

	labelAscent, labelHeight := lineMetrics(labelFace)
	labelBaseline := y + labelAscent
	drawTracked(dc, "МОЙ МЕСЯЦ НА ТАТАМИ", contentLeft, labelBaseline, labelSpacing)
	y += labelHeight

	y += gapLabelTitle

	titleFace := face(weightExtraBold, titleSize)
	dc.SetFontFace(titleFace)
	dc.SetColor(colorText)

	titleAscent, titleHeight := lineMetrics(titleFace)
	titleBaseline := y + titleAscent
	drawTracked(dc, monthTitle(month), contentLeft, titleBaseline, titleSpacing)
	titleBottom := y + titleHeight

	seal := scaleToHeight(sealImage, int(sealSize))
	sealW := float64(seal.Bounds().Dx())
	sealX := contentRight - sealW
	sealY := titleBaseline - sealSize
	dc.DrawImage(seal, int(math.Round(sealX)), int(math.Round(sealY)))

	return titleBottom
}

// drawTrainingsCount draws the big trainings number and its declined
// label on one baseline, returning the y just below it.
func drawTrainingsCount(dc *gg.Context, top float64, n int) float64 {
	const (
		gap      = 40.0
		numSize  = 140.0
		lblSize  = 28.0
		numToLbl = 20.0
	)

	y := top + gap

	numFace := face(weightExtraBold, numSize)
	dc.SetFontFace(numFace)
	numAscent, numHeight := lineMetrics(numFace)
	baseline := y + numAscent

	dc.SetColor(colorText)
	numStr := strconv.Itoa(n)
	dc.DrawString(numStr, contentLeft, baseline)
	numW, _ := dc.MeasureString(numStr)

	lblFace := face(weightSemiBold, lblSize)
	dc.SetFontFace(lblFace)
	dc.SetColor(colorSub)
	dc.DrawString(trainingsWord(n), contentLeft+numW+numToLbl, baseline)

	return y + numHeight
}

type numberBlock struct {
	n     int
	label string
}

// drawHoursAndRounds draws the hours block (always) and the rounds
// block (only when rounds > 0 — see the spec's own "no rounds, no
// block" rule), side by side, returning the y just below them.
func drawHoursAndRounds(dc *gg.Context, top float64, totalMinutes, rounds int32) float64 {
	const (
		gap      = 28.0
		numSize  = 72.0
		lblSize  = 22.0
		blockGap = 48.0
	)

	y := top + gap

	hours := roundHours(totalMinutes)

	blocks := []numberBlock{{hours, hoursWord(hours)}}
	if rounds > 0 {
		blocks = append(blocks, numberBlock{int(rounds), roundsWord(int(rounds))})
	}

	blockWidth := contentWidth
	if len(blocks) > 1 {
		blockWidth = (contentWidth - blockGap) / float64(len(blocks))
	}

	numFace := face(weightExtraBold, numSize)
	lblFace := face(weightSemiBold, lblSize)

	numAscent, numHeight := lineMetrics(numFace)
	lblAscent, lblHeight := lineMetrics(lblFace)

	numBaseline := y + numAscent
	labelTop := y + numHeight
	labelBaseline := labelTop + lblAscent

	for i, b := range blocks {
		x := contentLeft + float64(i)*(blockWidth+blockGap)

		dc.SetFontFace(numFace)
		dc.SetColor(colorText)
		dc.DrawString(strconv.Itoa(b.n), x, numBaseline)

		dc.SetFontFace(lblFace)
		dc.SetColor(colorSub)
		dc.DrawString(b.label, x, labelBaseline)
	}

	return labelTop + lblHeight
}

type typeSegment struct {
	label string
	count int
}

// drawTypeBar draws the proportional type-breakdown bar and its legend.
// A zero-count type is left out of both entirely (never a zero-width
// segment) — see the spec's own rule.
func drawTypeBar(dc *gg.Context, top float64, in Input) {
	const (
		gap            = 28.0
		barHeight      = 28.0
		barRadius      = 14.0
		barGap         = 6.0
		gapBarToLegend = 16.0
		legendSize     = 22.0
		legendGap      = 28.0
	)

	segs := typeSegments(in)
	if len(segs) == 0 {
		return
	}

	y := top + gap

	total := 0
	for _, s := range segs {
		total += s.count
	}

	available := contentWidth
	if len(segs) > 1 {
		available -= barGap * float64(len(segs)-1)
	}

	x := contentLeft

	for _, s := range segs {
		w := available * float64(s.count) / float64(total)

		dc.SetColor(typeColors[s.label])
		dc.DrawRoundedRectangle(x, y, w, barHeight, barRadius)
		dc.Fill()

		x += w + barGap
	}

	y += barHeight + gapBarToLegend

	legendFace := face(weightSemiBold, legendSize)
	dc.SetFontFace(legendFace)
	legendAscent, _ := lineMetrics(legendFace)
	baseline := y + legendAscent

	lx := contentLeft

	for _, s := range segs {
		dc.SetColor(typeColors[s.label])
		dc.DrawString(s.label, lx, baseline)

		labelW, _ := dc.MeasureString(s.label)

		countStr := " " + strconv.Itoa(s.count)
		dc.SetColor(colorSub)
		dc.DrawString(countStr, lx+labelW, baseline)

		countW, _ := dc.MeasureString(countStr)

		lx += labelW + countW + legendGap
	}
}

// typeSegments orders in's three type counts Gi, No-Gi, Open Mat — the
// fixed order the spec's own legend example uses — dropping any that are
// zero.
func typeSegments(in Input) []typeSegment {
	var segs []typeSegment

	if in.GiCount > 0 {
		segs = append(segs, typeSegment{"Gi", in.GiCount})
	}

	if in.NoGiCount > 0 {
		segs = append(segs, typeSegment{"No-Gi", in.NoGiCount})
	}

	if in.OpenMatCount > 0 {
		segs = append(segs, typeSegment{"Open Mat", in.OpenMatCount})
	}

	return segs
}

// drawFooter draws the top rule + handle + tagline, anchored to the
// canvas's own bottom margin (960 - marginBottom).
func drawFooter(dc *gg.Context) {
	const (
		lineThickness    = 4.0
		gapAfterLine     = 24.0
		handleSize       = 28.0
		taglineSize      = 20.0
		gapHandleTagline = 8.0
	)

	bottom := float64(logicalH) - marginBottom

	handleFace := face(weightExtraBold, handleSize)
	taglineFace := face(weightRegular, taglineSize)

	handleAscent, handleHeight := lineMetrics(handleFace)
	taglineAscent, taglineHeight := lineMetrics(taglineFace)

	taglineTop := bottom - taglineHeight
	handleBottom := taglineTop - gapHandleTagline
	handleTop := handleBottom - handleHeight
	lineTop := handleTop - gapAfterLine - lineThickness

	dc.SetColor(colorText)
	dc.DrawRoundedRectangle(contentLeft, lineTop, contentWidth, lineThickness, 0)
	dc.Fill()

	dc.SetFontFace(handleFace)
	dc.SetColor(colorText)
	dc.DrawString("@JitLogBot", contentLeft, handleTop+handleAscent)

	dc.SetFontFace(taglineFace)
	dc.SetColor(colorSub)
	dc.DrawString("Твой путь на татами под контролем", contentLeft, taglineTop+taglineAscent)
}
