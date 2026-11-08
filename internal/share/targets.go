package share

// Target cap-top/baseline positions, at the final 1080×1920 output —
// halved into this logical (540×960, outputScale=2) space. Every
// section is positioned directly against its own target (via
// baselineForCapTop), not by accumulating gaps down the page — see
// TestDebugLayoutPositions, which found the old gap-chain approach
// drifting 60-190px off these targets by the time it reached the
// bottom of the card.
const (
	targetLogoCapTop       = 270.0 / outputScale
	targetLabelCapTop      = 440.0 / outputScale
	targetTitleCapTop      = 490.0 / outputScale
	targetTrainingsCapTop  = 660.0 / outputScale
	targetHoursCapTop      = 900.0 / outputScale
	targetHoursLabelCapTop = 1030.0 / outputScale
	targetBarCapTop        = 1100.0 / outputScale
	targetLegendCapTop     = 1150.0 / outputScale
	targetFooterLineCapTop = 1370.0 / outputScale
	targetHandleCapTop     = 1410.0 / outputScale
	targetTaglineBaseline  = 1520.0 / outputScale
)
