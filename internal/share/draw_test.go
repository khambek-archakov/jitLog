package share

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFitNumberSize(t *testing.T) {
	t.Parallel()

	const base = 56.0 // internal/share's own hours/rounds numSize.

	t.Run("already fits — unchanged", func(t *testing.T) {
		t.Parallel()

		wide := measureString(weightExtraBold, base, "21") + 100
		assert.Equal(t, base, fitNumberSize(weightExtraBold, base, wide, "21"))
	})

	t.Run("shrinks in steps until it fits", func(t *testing.T) {
		t.Parallel()

		s := "4 800"
		full := measureString(weightExtraBold, base, s)

		// A width that the base size overflows but some smaller,
		// above-floor size fits.
		avail := full * 0.85

		got := fitNumberSize(weightExtraBold, base, avail, s)

		assert.Less(t, got, base)
		assert.GreaterOrEqual(t, got, base*0.7)
		assert.LessOrEqual(t, measureString(weightExtraBold, got, s), avail)
	})

	t.Run("never shrinks below 70%, even if it still doesn't fit", func(t *testing.T) {
		t.Parallel()

		got := fitNumberSize(weightExtraBold, base, 1, "4 800")
		assert.InDelta(t, base*0.7, got, 0.01)
	})
}
