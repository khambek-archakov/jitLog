package tgbotapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func TestToInlineKeyboard(t *testing.T) {
	t.Parallel()

	kb := dto.Keyboard{
		dto.Row(
			dto.Button{Label: "Callback", Data: "cb:1"},
			dto.Button{Label: "🔗 Link", URL: "https://example.com"},
		),
	}

	got := toInlineKeyboard(kb)

	require.Len(t, got.InlineKeyboard, 1)
	require.Len(t, got.InlineKeyboard[0], 2)

	cb := got.InlineKeyboard[0][0]
	assert.Equal(t, "Callback", cb.Text)
	require.NotNil(t, cb.CallbackData)
	assert.Equal(t, "cb:1", *cb.CallbackData)
	assert.Nil(t, cb.URL)

	link := got.InlineKeyboard[0][1]
	assert.Equal(t, "🔗 Link", link.Text)
	require.NotNil(t, link.URL)
	assert.Equal(t, "https://example.com", *link.URL)
	assert.Nil(t, link.CallbackData)
}
