package share

// Target block positions, at the final 1080×1920 output — halved into
// this logical (540×960, outputScale=2) space. Sourced directly from
// assets/share/reference/reference_layout.json (an HTML/CSS-rendered
// reference at the same canvas size) rather than hand-derived: its own
// box tops are CSS line-box tops, not tight glyph ink — a few px above
// where a given string's own cap-top actually falls, which is exactly
// the "6-8px" the reference's own author called a normal difference —
// see TestMatchesReference, which checks against it directly.
const (
	targetLogoCapTop       = 270.0 / outputScale
	targetLabelCapTop      = 438.0 / outputScale
	targetTitleCapTop      = 482.0 / outputScale
	targetSealTop          = 478.0 / outputScale
	targetTrainingsCapTop  = 642.0 / outputScale
	targetHoursCapTop      = 880.0 / outputScale
	targetHoursLabelCapTop = 985.0 / outputScale
	targetBarCapTop        = 1089.0 / outputScale
	targetLegendCapTop     = 1139.0 / outputScale
	targetFooterLineCapTop = 1420.0 / outputScale
	targetHandleCapTop     = 1460.0 / outputScale
	targetTaglineCapTop    = 1505.0 / outputScale
)
