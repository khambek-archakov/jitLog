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
	targetLogoCapTop      = 270.0 / outputScale
	targetLabelCapTop     = 438.0 / outputScale
	targetTitleCapTop     = 482.0 / outputScale
	targetSealTop         = 494.0 / outputScale
	targetTrainingsCapTop = 642.0 / outputScale
	targetHoursCapTop     = 880.0 / outputScale
	// targetHoursLabelCapTop was derived from hours_rounds_row's own
	// combined box (there's no separate label sub-box in the reference
	// JSON) — unaffected by the hours/rounds number growing to 126px,
	// since that's the same cap-top-pinned sizing every other block
	// uses (see baselineForCapTop): a bigger font shifts the number's
	// own baseline down, never the label's.
	targetHoursLabelCapTop = 985.0 / outputScale
	targetBarCapTop        = 1089.0 / outputScale
	targetLegendCapTop     = 1139.0 / outputScale
	// footer line/handle: originally 1420/1460, moved up 80px to
	// 1340/1380, then back down 40px to the values below (net 40px up
	// from the original) — the handle keeps the same 40px gap below the
	// line it's always had. The tagline that used to follow at 1465 is
	// gone; nothing draws below the handle any more (see drawFooter).
	targetFooterLineCapTop = 1380.0 / outputScale
	targetHandleCapTop     = 1420.0 / outputScale
)
