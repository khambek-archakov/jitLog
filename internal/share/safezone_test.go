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
