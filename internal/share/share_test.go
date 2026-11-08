package share

import (
	"bytes"
	"image"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender(t *testing.T) {
	t.Parallel()

	data, err := Render(Input{
		Name: "Алексей", Month: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Trainings: 14, TotalMinutes: 21 * 60, Rounds: 63,
		GiCount: 8, NoGiCount: 4, OpenMatCount: 2,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	img, format, err := image.Decode(bytes.NewReader(data))
	require.NoError(t, err)
	assert.Equal(t, "png", format)

	b := img.Bounds()
	assert.Equal(t, 1080, b.Dx())
	assert.Equal(t, 1920, b.Dy())
}

func TestRender_NoRounds_NoBlockOrSegment(t *testing.T) {
	t.Parallel()

	// Can't inspect pixels meaningfully here without a reference image —
	// this just guards that a single-type, zero-rounds month still
	// renders successfully (the real "no block/segment drawn" behavior
	// is exercised visually; see the package's own doc for the 3
	// generated sample images this was checked against).
	data, err := Render(Input{
		Name: "Мария", Month: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Trainings: 1, TotalMinutes: 60, Rounds: 0,
		GiCount: 1,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestRender_LongLegend(t *testing.T) {
	t.Parallel()

	data, err := Render(Input{
		Name: "Алексей", Month: time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC),
		Trainings: 329, TotalMinutes: 329 * 60, Rounds: 1200,
		GiCount: 128, NoGiCount: 104, OpenMatCount: 97,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}

func TestRender_LongNameAndLargeNumbers(t *testing.T) {
	t.Parallel()

	data, err := Render(Input{
		Name:      "Константин Александрович Долгопрудненский-Воронцов",
		Month:     time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC),
		Trainings: 31, TotalMinutes: 99 * 60, Rounds: 480,
		GiCount: 15, NoGiCount: 10, OpenMatCount: 6,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, data)
}
