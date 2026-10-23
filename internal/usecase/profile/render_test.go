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

	t.Run("no timezone set — no timezone line at all", func(t *testing.T) {
		t.Parallel()

		text := profileText(&model.User{Belt: model.BeltWhite})

		assert.NotContains(t, text, "Часовой пояс")
	})

	t.Run("timezone set shows the zone label — no reminders status, that's the schedule screen's job", func(t *testing.T) {
		t.Parallel()

		tz := "Asia/Yekaterinburg"

		text := profileText(&model.User{Belt: model.BeltWhite, Timezone: &tz, RemindersEnabled: true})

		assert.Contains(t, text, "Часовой пояс: Екатеринбург, Алматы +5")
		assert.NotContains(t, text, "Напоминания")
	})
}

func TestProfileKeyboard(t *testing.T) {
	t.Parallel()

	kb := profileKeyboard()

	require.Len(t, kb, 2)
	assert.Equal(t, "profile:edit", kb[0][0].Data)
	assert.Equal(t, "menu:back", kb[1][0].Data)
}

func TestTZPickerKeyboard(t *testing.T) {
	t.Parallel()

	kb := tzPickerKeyboard()

	require.Len(t, kb, 4)
	assert.Equal(t, "profile:tz:set:msk", kb[0][0].Data)
	assert.Equal(t, "profile:tz:set:sam", kb[1][0].Data)
	assert.Equal(t, "profile:tz:set:yek", kb[2][0].Data)
	// The back button reuses callbackProfileEdit directly — same pattern
	// as beltPickerKeyboard's own "← Назад" — no dedicated cancel route.
	assert.Equal(t, "profile:edit", kb[3][0].Data)
}

func TestEditMenuKeyboard(t *testing.T) {
	t.Parallel()

	kb := editMenuKeyboard()

	require.Len(t, kb, 4)
	assert.Equal(t, "profile:belt:edit", kb[0][0].Data)
	assert.Equal(t, "profile:age:edit", kb[1][0].Data)
	assert.Equal(t, "profile:tz:edit", kb[2][0].Data)
	assert.Equal(t, "profile:show", kb[3][0].Data)
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

	assert.Equal(t, "profile:edit", kb[4][0].Data)
}

func TestBeltRank(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 0, beltRank(model.BeltWhite))
	assert.Equal(t, 1, beltRank(model.BeltBlue))
	assert.Equal(t, 2, beltRank(model.BeltPurple))
	assert.Equal(t, 3, beltRank(model.BeltBrown))
	assert.Equal(t, 4, beltRank(model.BeltBlack))
	assert.Equal(t, -1, beltRank(model.BeltNone))
}

func TestBeltInstrumental(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "белым", beltInstrumental(model.BeltWhite))
	assert.Equal(t, "синим", beltInstrumental(model.BeltBlue))
	assert.Equal(t, "пурпурным", beltInstrumental(model.BeltPurple))
	assert.Equal(t, "коричневым", beltInstrumental(model.BeltBrown))
	assert.Equal(t, "чёрным", beltInstrumental(model.BeltBlack))
}

func TestBeltPrepositional(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "белом", beltPrepositional(model.BeltWhite))
	assert.Equal(t, "синем", beltPrepositional(model.BeltBlue))
	assert.Equal(t, "пурпурном", beltPrepositional(model.BeltPurple))
	assert.Equal(t, "коричневом", beltPrepositional(model.BeltBrown))
	assert.Equal(t, "чёрном", beltPrepositional(model.BeltBlack))
}

func TestTrainingWord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int
		want string
	}{
		{n: 1, want: "тренировка"},
		{n: 2, want: "тренировки"},
		{n: 4, want: "тренировки"},
		{n: 5, want: "тренировок"},
		{n: 11, want: "тренировок"},
		{n: 21, want: "тренировка"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, trainingWord(tc.n))
	}
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
