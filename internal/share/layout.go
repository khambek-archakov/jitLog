package share

import (
	"math"
	"strconv"
	"time"

	"github.com/fogleman/gg"
)

// drawSeigaihaBottom paints the bottom decorative wave pattern — the
// very first layer, before even the background color's own Clear has
// settled and well before drawDecor's own enso arc or any text. Unlike
// every other draw* function here, it must run before share.go's own
// dc.Scale(outputScale, outputScale) call: seigaihaBottomImage is
// already native final-pixel size (1080×440), so drawing it in that
// same pre-scale, untransformed coordinate space is what makes (0,1480)
// land exactly on final pixels without any resizing — the same trick
// scaleToHeight exists to avoid needing elsewhere, just taken from the
// other direction (this image needs no resize at all, so it skips the
// logical coordinate system entirely instead of being resized to fit
// it).
func drawSeigaihaBottom(dc *gg.Context) {
	const (
		x = 0.0
		y = 1480.0
	)

	dc.DrawImage(seigaihaBottomImage, int(x), int(y))
}

// drawDecor paints the background's own faint "enso" arc — behind every
// other element, drawn right after the flat background fill and before
// anything else. Its center sits off-canvas to the right on purpose
// (contentRight tops out at 492); only the arc's left-hand sliver ever
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

// drawHeader draws the logo + name row. logoHeight is 56 logical px
// (112 final) — unchanged by logo_final.png's own, wider replacement
// (1321×587, vs. the old asset's 1000×496): only the width that height
// scales to changed, not the height target itself. logoShiftX is gone
// too — the old -8px nudge existed only to compensate for the previous
// logo's own clipped left edge ("J"'s tail); logo_final.png's own left
// edge needs no such compensation, so the logo's own left edge now sits
// exactly on contentLeft, like every other block.
func drawHeader(dc *gg.Context, name string, tr *trace) {
	const (
		logoHeight  = 56.0
		nameGap     = 20.0
		nameSize    = 22.0  // 44 final.
		nameMaxWide = 300.0 // 600 final.
	)

	logo := scaleToHeight(logoImage, int(logoHeight))
	logoW := float64(logo.Bounds().Dx())

	logoX := contentLeft
	logoY := targetLogoCapTop

	dc.DrawImage(logo, int(math.Round(logoX)), int(math.Round(logoY)))
	tr.image("логотип", logoY)

	nameFace := face(weightSemiBold, nameSize)
	dc.SetFontFace(nameFace)
	dc.SetColor(colorText)

	maxNameWidth := math.Min(contentRight-(logoX+logoW)-nameGap, nameMaxWide)
	name = truncateToWidth(name, maxNameWidth, func(s string) float64 {
		w, _ := dc.MeasureString(s)
		return w
	})

	nameCenterY := logoY + logoHeight/2
	dc.DrawStringAnchored(name, contentRight, nameCenterY, 1, 0.5)
}

// drawTitle draws the period label ("Мой месяц на татами"/"Мой год на
// татами") and the month/year headline, each independently placed at
// its own target cap-top (from reference_layout.json), plus the seal
// (bottom aligned to the headline's own baseline, per the original spec
// — its own box top sits 4px above the title box's, consistent with
// "baseline" rather than "box bottom" alignment). The label no longer
// tracks letters out (that was tied to its old all-caps styling, since
// dropped) — only the headline still does, to keep longer month names
// ("Сентябрь 2026", "Декабрь 2026") clear of the seal in its own
// top-right corner.
func drawTitle(dc *gg.Context, p Period, anchor time.Time, tr *trace) {
	const (
		labelSize    = 16.0 // 32 final.
		titleSize    = 56.0 // 112 final.
		titleSpacing = -3.0
		sealSize     = 44.0 // 88 final.
	)

	labelText, title := periodTitle(p, anchor)

	labelFace := face(weightSemiBold, labelSize)
	dc.SetFontFace(labelFace)
	dc.SetColor(colorSub)

	labelBaseline := baselineForCapTop(weightSemiBold, labelSize, targetLabelCapTop, labelText)
	dc.DrawString(labelText, contentLeft, labelBaseline)
	tr.text("надпись", weightSemiBold, labelSize, labelBaseline, labelText)

	titleFace := face(weightSemiBold, titleSize)
	dc.SetFontFace(titleFace)
	dc.SetColor(colorText)

	titleBaseline := baselineForCapTop(weightSemiBold, titleSize, targetTitleCapTop, title)
	drawTracked(dc, title, contentLeft, titleBaseline, titleSpacing)
	tr.text("заголовок", weightSemiBold, titleSize, titleBaseline, title)

	seal := scaleToHeight(sealImage, int(sealSize))
	sealW := float64(seal.Bounds().Dx())
	sealX := contentRight - sealW
	sealY := titleBaseline - sealSize
	dc.DrawImage(seal, int(math.Round(sealX)), int(math.Round(sealY)))
}

// drawTrainingsCount draws the big trainings number and its declined
// label on one baseline, the number's own cap-top pinned to
// targetTrainingsCapTop. The number's own font shrinks (never the
// label's) in fitNumberSize's own steps if contentWidth minus the
// label's width can't otherwise fit it — a year's trainings count can
// run into three digits where a month's never does.
func drawTrainingsCount(dc *gg.Context, n int, tr *trace) {
	const (
		numSize  = 112.0 // 224 final.
		lblSize  = 16.0  // 32 final.
		numToLbl = 20.0
	)

	numStr := formatThousands(n)
	label := trainingsWord(n)

	avail := contentWidth - numToLbl - measureString(weightSemiBold, lblSize, label)
	size := fitNumberSize(weightExtraBold, numSize, avail, numStr)

	dc.SetFontFace(face(weightExtraBold, size))

	baseline := baselineForCapTop(weightExtraBold, size, targetTrainingsCapTop, numStr)

	dc.SetColor(colorText)
	dc.DrawString(numStr, contentLeft, baseline)
	tr.text("число тренировок", weightExtraBold, size, baseline, numStr)
	numW, _ := dc.MeasureString(numStr)

	dc.SetFontFace(face(weightSemiBold, lblSize))
	dc.SetColor(colorSub)
	dc.DrawString(label, contentLeft+numW+numToLbl, baseline)
}

type numberBlock struct {
	n     int
	label string
}

// drawHoursAndRounds draws the hours block (always) and the rounds
// block (only when rounds > 0 — see the spec's own "no rounds, no
// block" rule), side by side — each number's cap-top pinned to
// targetHoursCapTop, each label's cap-top pinned to
// targetHoursLabelCapTop. Each number's own font shrinks independently
// (fitNumberSize, against a fixed half-row-minus-gap budget, not the
// natural layout width below) if it would otherwise run past that
// budget — a year's hours/rounds totals can run into four digits where
// a month's never do.
func drawHoursAndRounds(dc *gg.Context, totalMinutes, rounds int32, tr *trace) {
	const (
		numSize  = 56.0 // 112 final.
		lblSize  = 16.0 // 32 final.
		blockGap = 48.0

		// shrinkGap is the spec's own fixed clearance (80 final) a
		// number must leave within half of contentWidth before it
		// starts shrinking — independent of blockGap/the natural
		// content-hugging width below, which is what actually
		// positions the drawn blocks.
		shrinkGap = 40.0 // 80 final.
	)

	shrinkAvail := contentWidth/2 - shrinkGap

	hours := roundHours(totalMinutes)

	blocks := []numberBlock{{hours, hoursWord(hours)}}
	if rounds > 0 {
		blocks = append(blocks, numberBlock{int(rounds), roundsWord(int(rounds))})
	}

	lblFace := face(weightSemiBold, lblSize)

	blockNames := []string{"часы", "раунды"}

	// Blocks hug their own content (number or label, whichever is wider)
	// with a fixed gap between them — not split into even columns across
	// contentWidth. The reference's own "21"/"63" sit close together on
	// the left, not spread across the full row (confirmed by scanning
	// assets/share/reference/reference_full.png directly: "63" starts
	// ~86px/final after "21" ends, matching "21"'s own measured width
	// plus blockGap, not a half-row split).
	x := contentLeft

	for i, b := range blocks {
		numStr := formatThousands(b.n)
		size := fitNumberSize(weightExtraBold, numSize, shrinkAvail, numStr)
		numBaseline := baselineForCapTop(weightExtraBold, size, targetHoursCapTop, numStr)

		dc.SetFontFace(face(weightExtraBold, size))
		dc.SetColor(colorText)
		dc.DrawString(numStr, x, numBaseline)
		tr.text(blockNames[i]+" (число)", weightExtraBold, size, numBaseline, numStr)
		numW, _ := dc.MeasureString(numStr)

		labelBaseline := baselineForCapTop(weightSemiBold, lblSize, targetHoursLabelCapTop, b.label)

		dc.SetFontFace(lblFace)
		dc.SetColor(colorSub)
		dc.DrawString(b.label, x, labelBaseline)

		if i == 0 {
			tr.text("часы/раунды (подпись)", weightSemiBold, lblSize, labelBaseline, b.label)
		}

		lblW, _ := dc.MeasureString(b.label)
		x += math.Max(numW, lblW) + blockGap
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
		barHeight = 14.0 // 28 final.
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

// Legend sizing — 32px final, i.e. 16 here, matching the card's own
// shared secondary-text size bucket (label/unit-captions/legend, all
// #4A4F5A). legendGapFloor/legendSizeFloor are the shrink ladder's
// own two rungs, tried in order, before falling back to wrapping the
// last item onto its own second line — see drawLegend's own doc comment
// for why each step exists.
const (
	legendSizeDefault = 16.0 // 32 final.
	legendGapDefault  = 14.0 // 28 final.
	swatchSize        = 9.0  // 18 final.
	swatchGap         = 5.0  // 10 final.
	swatchRadius      = swatchSize * 0.2

	legendGapFloor  = 10.0 // 20 final.
	legendSizeFloor = 12.0 // 24 final.

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

		dc.SetColor(colorSub)
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

// drawFooter draws the top rule (targetFooterLineCapTop) and the handle
// (targetHandleCapTop) — nothing is drawn below the handle any more
// (the tagline that used to sit there is gone); the line and handle
// themselves keep their own existing positions untouched.
func drawFooter(dc *gg.Context, tr *trace) {
	const (
		lineThickness = 4.0
		handleSize    = 22.0 // 44 final.
	)

	lineTop := targetFooterLineCapTop

	dc.SetColor(colorText)
	dc.DrawRoundedRectangle(contentLeft, lineTop, contentWidth, lineThickness, 0)
	dc.Fill()
	tr.shape("линия футера", lineTop)

	handleFace := face(weightSemiBold, handleSize)
	dc.SetFontFace(handleFace)
	dc.SetColor(colorText)

	const handleText = "@JitLogBot"
	handleBaseline := baselineForCapTop(weightSemiBold, handleSize, targetHandleCapTop, handleText)
	dc.DrawString(handleText, contentLeft, handleBaseline)
	tr.text("@JitLogBot", weightSemiBold, handleSize, handleBaseline, handleText)
}
