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
// shows. Exempt from the safe-zone/trace checks — it's decor, not
// content.
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
	dc.SetRGBA255(0x16, 0x18, 0x1D, int(math.Round(255*0.04)))

	a1 := startDeg * math.Pi / 180
	a2 := (startDeg + sweepDeg) * math.Pi / 180

	dc.DrawArc(cx, cy, radius, a1, a2)
	dc.Stroke()
}

// drawHeader draws the logo + name row. logoHeight is 64 logical px —
// the spec's own "128 px" is stated at the final 1080×1920 output, i.e.
// half that in this scaled-×2 logical space (see share.go's own
// outputScale). The logo's own top sits exactly at targetLogoCapTop —
// the one block whose target already matched marginTop with no
// adjustment needed.
func drawHeader(dc *gg.Context, name string, tr *trace) {
	const (
		logoHeight = 64.0
		logoShiftX = -8.0
		nameGap    = 20.0
		nameSize   = 30.0
	)

	logo := scaleToHeight(logoImage, int(logoHeight))
	logoW := float64(logo.Bounds().Dx())

	logoX := contentLeft + logoShiftX
	logoY := targetLogoCapTop

	dc.DrawImage(logo, int(math.Round(logoX)), int(math.Round(logoY)))
	tr.image("логотип", logoY)

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
}

// drawTitle draws "МОЙ МЕСЯЦ НА ТАТАМИ" and the month/year headline, each
// independently placed at its own target cap-top, plus the seal (bottom
// aligned to the headline's own baseline, per the original spec).
func drawTitle(dc *gg.Context, month time.Time, tr *trace) {
	const (
		labelSize    = 24.0
		labelSpacing = 3.0
		titleSize    = 36.0
		titleSpacing = -1.0
		sealSize     = 36.0
	)

	labelFace := face(weightSemiBold, labelSize)
	dc.SetFontFace(labelFace)
	dc.SetColor(colorAccent)

	const labelText = "МОЙ МЕСЯЦ НА ТАТАМИ"
	labelBaseline := baselineForCapTop(weightSemiBold, labelSize, targetLabelCapTop, labelText)
	drawTracked(dc, labelText, contentLeft, labelBaseline, labelSpacing)
	tr.text("надпись", weightSemiBold, labelSize, labelBaseline, labelText)

	titleFace := face(weightExtraBold, titleSize)
	dc.SetFontFace(titleFace)
	dc.SetColor(colorText)

	title := monthTitle(month)
	titleBaseline := baselineForCapTop(weightExtraBold, titleSize, targetTitleCapTop, title)
	drawTracked(dc, title, contentLeft, titleBaseline, titleSpacing)
	tr.text("заголовок", weightExtraBold, titleSize, titleBaseline, title)

	seal := scaleToHeight(sealImage, int(sealSize))
	sealW := float64(seal.Bounds().Dx())
	sealX := contentRight - sealW
	sealY := titleBaseline - sealSize
	dc.DrawImage(seal, int(math.Round(sealX)), int(math.Round(sealY)))
}

// drawTrainingsCount draws the big trainings number and its declined
// label on one baseline, the number's own cap-top pinned to
// targetTrainingsCapTop.
func drawTrainingsCount(dc *gg.Context, n int, tr *trace) {
	const (
		// numSize was 140 — shrunk so the number's own baseline also
		// lands on targetTrainingsCapTop's own ascent target (824 final),
		// not just its cap-top; see TestDebugLayoutPositions.
		numSize  = 112.0
		lblSize  = 28.0
		numToLbl = 20.0
	)

	numFace := face(weightExtraBold, numSize)
	dc.SetFontFace(numFace)

	numStr := strconv.Itoa(n)
	baseline := baselineForCapTop(weightExtraBold, numSize, targetTrainingsCapTop, numStr)

	dc.SetColor(colorText)
	dc.DrawString(numStr, contentLeft, baseline)
	tr.text("число тренировок", weightExtraBold, numSize, baseline, numStr)
	numW, _ := dc.MeasureString(numStr)

	lblFace := face(weightSemiBold, lblSize)
	dc.SetFontFace(lblFace)
	dc.SetColor(colorSub)
	dc.DrawString(trainingsWord(n), contentLeft+numW+numToLbl, baseline)
}

type numberBlock struct {
	n     int
	label string
}

// drawHoursAndRounds draws the hours block (always) and the rounds
// block (only when rounds > 0 — see the spec's own "no rounds, no
// block" rule), side by side — each number's cap-top pinned to
// targetHoursCapTop, each label's cap-top pinned to
// targetHoursLabelCapTop.
func drawHoursAndRounds(dc *gg.Context, totalMinutes, rounds int32, tr *trace) {
	const (
		// numSize was 72 — shrunk for the same reason as
		// drawTrainingsCount's own numSize, to also hit the baseline
		// target (982 final), not just cap-top.
		numSize  = 55.0
		lblSize  = 22.0
		blockGap = 48.0
	)

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

	blockNames := []string{"часы", "раунды"}

	for i, b := range blocks {
		x := contentLeft + float64(i)*(blockWidth+blockGap)

		numStr := strconv.Itoa(b.n)
		numBaseline := baselineForCapTop(weightExtraBold, numSize, targetHoursCapTop, numStr)

		dc.SetFontFace(numFace)
		dc.SetColor(colorText)
		dc.DrawString(numStr, x, numBaseline)
		tr.text(blockNames[i]+" (число)", weightExtraBold, numSize, numBaseline, numStr)

		labelBaseline := baselineForCapTop(weightSemiBold, lblSize, targetHoursLabelCapTop, b.label)

		dc.SetFontFace(lblFace)
		dc.SetColor(colorSub)
		dc.DrawString(b.label, x, labelBaseline)

		if i == 0 {
			tr.text("часы/раунды (подпись)", weightSemiBold, lblSize, labelBaseline, b.label)
		}
	}
}

type typeSegment struct {
	label string
	count int
}

// drawTypeBar draws the proportional type-breakdown bar (top pinned to
// targetBarCapTop) and its legend (top pinned to targetLegendCapTop),
// returning the legend's own bottom y (top, unchanged, if there's
// nothing to draw) — used only to verify clearance from the footer (see
// TestLegendClearsFooter), since every block's own position is now
// independent of what was drawn above it. A zero-count type is left out
// of both entirely (never a zero-width segment) — see the spec's own
// rule.
func drawTypeBar(dc *gg.Context, in Input, tr *trace) float64 {
	const (
		// barHeight was 28 (logical) — the user's own "полоса 1100
		// (высота 28)" note gives 28 at final scale, same as every other
		// number in that message; at 28 logical (56 final) the bar's own
		// bottom (1100+56=1156 final) ran past targetLegendCapTop (1150),
		// overlapping the legend above it.
		barHeight = 14.0
		barRadius = 7.0
		barGap    = 6.0
	)

	segs := typeSegments(in)
	if len(segs) == 0 {
		return targetBarCapTop
	}

	y := targetBarCapTop

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

	tr.shape("полоса", y)

	return drawLegend(dc, segs, tr)
}

// Legend sizing — the spec's own "28/28/18/10" are given at the final
// 1080×1920 output, hence the /2 here (see share.go's outputScale).
// legendGapFloor/legendSizeFloor are the shrink ladder's own two rungs,
// tried in order, before falling back to wrapping the last item onto its
// own second line — see drawLegend's own doc comment for why each step
// exists.
const (
	legendSizeDefault = 14.0 // 28 final
	legendGapDefault  = 14.0 // 28 final
	swatchSize        = 9.0  // 18 final
	swatchGap         = 5.0  // 10 final
	swatchRadius      = swatchSize * 0.2

	legendGapFloor  = 10.0 // 20 final
	legendSizeFloor = 12.0 // 24 final

	legendRowGap = 8.0 // vertical gap when the last item wraps to its own line.
)

// drawLegend lays segs out left to right, shrinking first the gap
// between items, then the font size, if the default sizing would run
// past contentRight — and, failing that, wraps the last item onto its
// own second line. Every count is unbounded user data (a very active
// month can easily reach triple digits), so this has to hold at any
// width, not just the sizes this feature happened to be designed around.
// The first row's own cap-top is pinned to targetLegendCapTop; a second
// (wrapped) row follows it by legendRowGap.
func drawLegend(dc *gg.Context, segs []typeSegment, tr *trace) float64 {
	size, gap := legendSizeDefault, legendGapDefault

	if legendWidth(dc, segs, size, gap) > contentWidth {
		gap = legendGapFloor
	}

	if legendWidth(dc, segs, size, gap) > contentWidth {
		size = legendSizeFloor
	}

	if legendWidth(dc, segs, size, gap) > contentWidth && len(segs) > 1 {
		rowBottom := drawLegendRow(dc, segs[:len(segs)-1], targetLegendCapTop, size, gap, tr)

		return drawLegendRowAtCapTop(dc, segs[len(segs)-1:], rowBottom+legendRowGap, size, gap, nil)
	}

	return drawLegendRow(dc, segs, targetLegendCapTop, size, gap, tr)
}

// legendWidth is drawLegendRow's own total width, computed without
// drawing anything.
func legendWidth(dc *gg.Context, segs []typeSegment, size, gap float64) float64 {
	dc.SetFontFace(face(weightSemiBold, size))

	var w float64

	for i, s := range segs {
		w += swatchSize + swatchGap

		labelW, _ := dc.MeasureString(s.label)
		countW, _ := dc.MeasureString(countSuffix(s.count))
		w += labelW + countW

		if i < len(segs)-1 {
			w += gap
		}
	}

	return w
}

// drawLegendRow draws one row of segs with its own cap-top pinned to
// capTop (via the row's first segment), returning the row's own bottom
// (capTop + ink height) for the wrap case's second row to follow.
func drawLegendRow(dc *gg.Context, segs []typeSegment, capTop, size, gap float64, tr *trace) float64 {
	legendFace := face(weightSemiBold, size)
	dc.SetFontFace(legendFace)

	baseline := baselineForCapTop(weightSemiBold, size, capTop, segs[0].label)
	_, maxY := inkBounds(weightSemiBold, size, segs[0].label)
	inkBottom := baseline + maxY
	swatchY := capTop + (inkBottom-capTop-swatchSize)/2

	lx := contentLeft

	for i, s := range segs {
		dc.SetColor(typeColors[s.label])
		dc.DrawRoundedRectangle(lx, swatchY, swatchSize, swatchSize, swatchRadius)
		dc.Fill()

		lx += swatchSize + swatchGap

		dc.SetColor(colorText)
		dc.DrawString(s.label, lx, baseline)

		if i == 0 {
			tr.text("легенда", weightSemiBold, size, baseline, s.label)
		}

		labelW, _ := dc.MeasureString(s.label)

		countStr := countSuffix(s.count)
		dc.SetColor(colorSub)
		dc.DrawString(countStr, lx+labelW, baseline)

		countW, _ := dc.MeasureString(countStr)

		lx += labelW + countW + gap
	}

	return baseline + maxY
}

// drawLegendRowAtCapTop is drawLegendRow's own twin for the wrap case's
// second row, where the caller already computed its own top (not a
// target constant) — kept as a separate name purely for readability at
// the call site.
func drawLegendRowAtCapTop(dc *gg.Context, segs []typeSegment, capTop, size, gap float64, tr *trace) float64 {
	return drawLegendRow(dc, segs, capTop, size, gap, tr)
}

func countSuffix(n int) string {
	return " " + strconv.Itoa(n)
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

// drawFooter draws the top rule (targetFooterLineCapTop), the handle
// (targetHandleCapTop) and the tagline — the tagline is the one block
// the spec gave a baseline target for directly (targetTaglineBaseline),
// rather than a cap-top, so it's used as-is, no BoundString needed.
func drawFooter(dc *gg.Context, tr *trace) {
	const (
		lineThickness = 4.0
		handleSize    = 28.0
		taglineSize   = 20.0
	)

	lineTop := targetFooterLineCapTop

	dc.SetColor(colorText)
	dc.DrawRoundedRectangle(contentLeft, lineTop, contentWidth, lineThickness, 0)
	dc.Fill()
	tr.shape("линия футера", lineTop)

	handleFace := face(weightExtraBold, handleSize)
	dc.SetFontFace(handleFace)
	dc.SetColor(colorText)

	const handleText = "@JitLogBot"
	handleBaseline := baselineForCapTop(weightExtraBold, handleSize, targetHandleCapTop, handleText)
	dc.DrawString(handleText, contentLeft, handleBaseline)
	tr.text("@JitLogBot", weightExtraBold, handleSize, handleBaseline, handleText)

	taglineFace := face(weightRegular, taglineSize)
	dc.SetFontFace(taglineFace)
	dc.SetColor(colorSub)

	const taglineText = "Твой путь на татами под контролем"
	taglineBaseline := targetTaglineBaseline
	dc.DrawString(taglineText, contentLeft, taglineBaseline)
	tr.text("таглайн", weightRegular, taglineSize, taglineBaseline, taglineText)
}
