package profile

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/model"
)

func TestProfileText(t *testing.T) {
	t.Parallel()

	t.Run("with name and age", func(t *testing.T) {
		t.Parallel()

		name := "test"
		age := int16(28)

		text := profileText(&model.User{Name: &name, Age: &age, Belt: model.BeltBrown})

		assert.Contains(t, text, "Имя: test")
		assert.Contains(t, text, "Возраст: 28")
		assert.Contains(t, text, "🟤 Коричневый")
	})

	t.Run("missing age falls back to a dash", func(t *testing.T) {
		t.Parallel()

		name := "test"

		text := profileText(&model.User{Name: &name, Belt: model.BeltWhite})

		assert.Contains(t, text, "Возраст: —")
	})
}

func TestBeltPickerKeyboard(t *testing.T) {
	t.Parallel()

	kb := beltPickerKeyboard(model.BeltBlue)

	require.Len(t, kb, 5) // 5 belts - current + back row

	var labels []string
	for _, row := range kb[:4] {
		labels = append(labels, row[0].Label)
	}

	assert.NotContains(t, labels, beltLabel(model.BeltBlue))
	assert.Contains(t, labels, beltLabel(model.BeltWhite))
	assert.Contains(t, labels, beltLabel(model.BeltPurple))
	assert.Contains(t, labels, beltLabel(model.BeltBrown))
	assert.Contains(t, labels, beltLabel(model.BeltBlack))

	assert.Equal(t, "profile:show", kb[4][0].Data)
}

func TestBeltTokenRoundTrip(t *testing.T) {
	t.Parallel()

	belts := []model.Belt{model.BeltWhite, model.BeltBlue, model.BeltPurple, model.BeltBrown, model.BeltBlack}

	for _, b := range belts {
		got, ok := beltFromToken(beltToken(b))
		require.True(t, ok)
		assert.Equal(t, b, got)
	}

	_, ok := beltFromToken("bogus")
	assert.False(t, ok)
}
