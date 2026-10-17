package winback

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWinbackKeyboard(t *testing.T) {
	t.Parallel()

	kb := winbackKeyboard()

	require.Len(t, kb, 2)
	require.Len(t, kb[0], 1)
	require.Len(t, kb[1], 1)

	assert.Equal(t, "Привет 👋", kb[0][0].Label)
	assert.Equal(t, callbackHi, kb[0][0].Data)

	assert.Equal(t, "Вот давай без гифок", kb[1][0].Label)
	assert.Equal(t, callbackOff, kb[1][0].Data)
}
