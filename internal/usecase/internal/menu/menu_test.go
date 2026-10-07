package menu_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/usecase/internal/menu"
)

func TestKeyboard(t *testing.T) {
	t.Parallel()

	kb := menu.Keyboard()

	require.Len(t, kb, 6)
	assert.Equal(t, "menu:add_training", kb[0][0].Data)
	assert.Equal(t, "training:history:page:0", kb[1][0].Data)
	assert.Equal(t, "schedule:list", kb[2][0].Data)
	assert.Equal(t, "competition:list", kb[3][0].Data)
	assert.Equal(t, "stats:period:week", kb[4][0].Data)
	assert.Equal(t, "profile:show", kb[5][0].Data)
}
