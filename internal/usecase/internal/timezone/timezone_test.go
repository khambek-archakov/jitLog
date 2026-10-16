package timezone_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/usecase/internal/timezone"
)

func TestByCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code string
		iana string
	}{
		{code: "msk", iana: "Europe/Moscow"},
		{code: "sam", iana: "Europe/Samara"},
		{code: "yek", iana: "Asia/Yekaterinburg"},
	}

	for _, tc := range tests {
		z, ok := timezone.ByCode(tc.code)
		require.True(t, ok)
		assert.Equal(t, tc.iana, z.IANA)
	}

	t.Run("unrecognized code is rejected", func(t *testing.T) {
		t.Parallel()

		_, ok := timezone.ByCode("utc")
		assert.False(t, ok)

		_, ok = timezone.ByCode("")
		assert.False(t, ok)
	})

	t.Run("kaliningrad was removed from the list", func(t *testing.T) {
		t.Parallel()

		_, ok := timezone.ByCode("kgd")
		assert.False(t, ok)
	})
}

func TestLabelForIANA(t *testing.T) {
	t.Parallel()

	label, ok := timezone.LabelForIANA("Europe/Moscow")
	require.True(t, ok)
	assert.Equal(t, "Москва +3", label)

	_, ok = timezone.LabelForIANA("Europe/London")
	assert.False(t, ok)

	t.Run("kaliningrad has no label — it was removed from the list", func(t *testing.T) {
		t.Parallel()

		_, ok := timezone.LabelForIANA("Europe/Kaliningrad")
		assert.False(t, ok)
	})
}

func TestRows(t *testing.T) {
	t.Parallel()

	kb := timezone.Rows("schedule:tz:")

	require.Len(t, kb, 3)
	require.Len(t, kb[0], 1)
	require.Len(t, kb[1], 1)
	require.Len(t, kb[2], 1)

	assert.Equal(t, "schedule:tz:msk", kb[0][0].Data)
	assert.Equal(t, "schedule:tz:sam", kb[1][0].Data)
	assert.Equal(t, "schedule:tz:yek", kb[2][0].Data)
}
