package share

import (
	"bytes"
	"image"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// safeZoneTop/safeZoneBottom are Instagram/Telegram's own story UI safe
// zone, at the final 1080×1920 output — content (text or images; the
// enso decor is explicitly exempt, see drawDecor's own doc comment)
// must never draw outside [270, 1536].
const (
	safeZoneTop    = 270
	safeZoneBottom = 1536
)

// safeZoneLeft/safeZoneRight are contentLeft/contentRight at final
// scale — every row's own content, including the header (logo + name),
// must stay within them.
const (
	safeZoneLeft  = 96
	safeZoneRight = 984
)

// contentColorDistance is how far a pixel's color must sit from colorBg
// before it counts as "content" rather than background or the faint (4%
// opacity) enso decor — chosen well above what 4% opacity can ever
// produce (see the computation in this test's own doc) and well below
// any full-opacity text/image pixel, including anti-aliased edges.
const contentColorDistance = 60

// TestContentStaysInSafeZone renders every shipped sample and fails if
// any non-background, non-decor pixel falls outside [safeZoneTop,
// safeZoneBottom] — a direct, black-box check of the actual PNG output,
// not of the drawing code's own internal bookkeeping (which is exactly
// what let the old margins quietly drift outside the safe zone in the
// first place).
func TestContentStaysInSafeZone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Input
	}{
		{
			name: "three types, with a rounds block — the densest shipped layout",
			in: Input{
				Name: "Алексей", Month: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
				Trainings: 14, TotalMinutes: 21 * 60, Rounds: 63,
				GiCount: 8, NoGiCount: 4, OpenMatCount: 2,
			},
		},
		{
			name: "one type, no rounds block",
			in: Input{
				Name: "Мария", Month: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
				Trainings: 1, TotalMinutes: 60, Rounds: 0,
				GiCount: 1,
			},
		},
		{
			name: "long name and large numbers",
			in: Input{
				Name:      "Константин Александрович Долгопрудненский-Воронцов",
				Month:     time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
				Trainings: 31, TotalMinutes: 99 * 60, Rounds: 480,
				GiCount: 15, NoGiCount: 10, OpenMatCount: 6,
			},
		},
		{
			name: "long legend, wrapped",
			in: Input{
				Name: "Алексей", Month: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
				Trainings: 329, TotalMinutes: 329 * 60, Rounds: 1200,
				GiCount: 128, NoGiCount: 104, OpenMatCount: 97,
			},
		},
		{
			name: "year: three types, large numbers",
			in: Input{
				Period: PeriodYear,
				Name:   "Алексей", Month: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				Trainings: 187, TotalMinutes: 312 * 60, Rounds: 1204,
				GiCount: 110, NoGiCount: 60, OpenMatCount: 17,
			},
		},
		{
			name: "year: one type, no rounds",
			in: Input{
				Period: PeriodYear,
				Name:   "Алексей", Month: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				Trainings: 4, TotalMinutes: 6 * 60, Rounds: 0,
				NoGiCount: 4,
			},
		},
		{
			name: "year: long name, very large numbers",
			in: Input{
				Period:    PeriodYear,
				Name:      "Константин Александрович Долгопрудненский-Воронцов",
				Month:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				Trainings: 365, TotalMinutes: 999 * 60, Rounds: 4800,
				GiCount: 200, NoGiCount: 100, OpenMatCount: 65,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, err := Render(tc.in)
			require.NoError(t, err)

			top, bottom, found := contentYExtent(t, data)
			require.True(t, found, "no content pixels found at all")

			assert.GreaterOrEqualf(t, top, safeZoneTop, "content starts at y=%d, above the safe zone's own top (%d)", top, safeZoneTop)
			assert.LessOrEqualf(t, bottom, safeZoneBottom, "content ends at y=%d, below the safe zone's own bottom (%d)", bottom, safeZoneBottom)

			left, right, foundX := contentXExtent(t, data, 0)
			require.True(t, foundX, "no content pixels found at all")

			assert.GreaterOrEqualf(t, left, safeZoneLeft, "content starts at x=%d, left of the safe zone's own left edge (%d)", left, safeZoneLeft)
			assert.LessOrEqualf(t, right, safeZoneRight, "content ends at x=%d, right of the safe zone's own right edge (%d)", right, safeZoneRight)
		})
	}
}

// handleSizeForTest mirrors drawFooter's own unexported handleSize
// constant, purely so TestFooterEndsAtHandle can compute "@JitLogBot"'s
// own true ink bottom the same way drawFooter positions it.
const handleSizeForTest = 22.0 // 44 final — keep in sync with drawFooter's own handleSize.

// TestFooterEndsAtHandle guards the tagline's removal: "@JitLogBot" is
// now the footer's own last line, and nothing may be recorded or drawn
// below its own true ink bottom.
func TestFooterEndsAtHandle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   Input
	}{
		{name: "month", in: Input{
			Name: "Алексей", Month: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
			Trainings: 14, TotalMinutes: 21 * 60, Rounds: 63,
			GiCount: 8, NoGiCount: 4, OpenMatCount: 2,
		}},
		{name: "year", in: Input{
			Period: PeriodYear,
			Name:   "Алексей", Month: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			Trainings: 187, TotalMinutes: 312 * 60, Rounds: 1204,
			GiCount: 110, NoGiCount: 60, OpenMatCount: 17,
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data, tr, err := renderTraced(tc.in, &trace{})
			require.NoError(t, err)

			var handle *blockPosition

			for i, b := range tr.blocks {
				assert.NotEqualf(t, "таглайн", b.name, "the tagline block should no longer be drawn at all")

				if b.name == "@JitLogBot" {
					handle = &tr.blocks[i]
				}
			}

			require.NotNil(t, handle, "no @JitLogBot block recorded")

			_, maxY := inkBounds(weightSemiBold, handleSizeForTest, "@JitLogBot")
			wantBottom := handle.baseline + maxY*outputScale

			_, bottom, found := contentYExtent(t, data)
			require.True(t, found, "no content pixels found at all")

			assert.InDeltaf(t, wantBottom, float64(bottom), 8,
				"content ends at y=%d, want close to @JitLogBot's own bottom (%.1f) — something is drawing below the footer handle",
				bottom, wantBottom,
			)
		})
	}
}

// contentYExtent scans png for the topmost and bottommost row containing
// at least one pixel whose color differs from colorBg by more than
// contentColorDistance — i.e. real text/image content, not the
// background or the faint enso decor.
func contentYExtent(t *testing.T, png []byte) (top, bottom int, found bool) {
	t.Helper()

	img, _, err := image.Decode(bytes.NewReader(png))
	require.NoError(t, err)

	bgR, bgG, bgB, _ := colorBg.RGBA()
	bgR, bgG, bgB = bgR>>8, bgG>>8, bgB>>8

	b := img.Bounds()
	top, bottom, found = -1, -1, false

	for y := b.Min.Y; y < b.Max.Y; y++ {
		rowHasContent := false

		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			dr := int(r>>8) - int(bgR)
			dg := int(g>>8) - int(bgG)
			db := int(bl>>8) - int(bgB)

			if dr*dr+dg*dg+db*db > contentColorDistance*contentColorDistance {
				rowHasContent = true

				break
			}
		}

		if rowHasContent {
			if !found {
				top = y
				found = true
			}

			bottom = y
		}
	}

	return top, bottom, found
}

// contentXExtent is contentYExtent's own twin for the horizontal axis,
// optionally restricted to rows at or below fromY (0 scans the whole
// image).
func contentXExtent(t *testing.T, png []byte, fromY int) (left, right int, found bool) {
	t.Helper()

	img, _, err := image.Decode(bytes.NewReader(png))
	require.NoError(t, err)

	bgR, bgG, bgB, _ := colorBg.RGBA()
	bgR, bgG, bgB = bgR>>8, bgG>>8, bgB>>8

	b := img.Bounds()
	left, right, found = -1, -1, false

	startY := b.Min.Y
	if fromY > startY {
		startY = fromY
	}

	for x := b.Min.X; x < b.Max.X; x++ {
		colHasContent := false

		for y := startY; y < b.Max.Y; y++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			dr := int(r>>8) - int(bgR)
			dg := int(g>>8) - int(bgG)
			db := int(bl>>8) - int(bgB)

			if dr*dr+dg*dg+db*db > contentColorDistance*contentColorDistance {
				colHasContent = true

				break
			}
		}

		if colHasContent {
			if !found {
				left = x
				found = true
			}

			right = x
		}
	}

	return left, right, found
}
