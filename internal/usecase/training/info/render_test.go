package info_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/info"
)

func TestBody(t *testing.T) {
	t.Parallel()

	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)

	t.Run("with notes", func(t *testing.T) {
		t.Parallel()

		notes := "armbar from closed guard"
		body := info.Body(&model.Training{
			Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 61, Notes: &notes,
		})

		assert.Contains(t, body, "15 марта 2026")
		assert.Contains(t, body, "🥋 Gi")
		assert.Contains(t, body, "61 минута")
		assert.Contains(t, body, "📝 armbar from closed guard")
	})

	t.Run("without notes", func(t *testing.T) {
		t.Parallel()

		body := info.Body(&model.Training{Date: date, TrainingType: model.TrainingTypeOpenMat, DurationMinutes: 90})

		assert.Contains(t, body, "🤼 Open Mat")
		assert.Contains(t, body, "90 минут")
		assert.NotContains(t, body, "📝")
	})

	t.Run("with rounds", func(t *testing.T) {
		t.Parallel()

		rounds := int16(5)
		body := info.Body(&model.Training{
			Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 60, Rounds: &rounds,
		})

		assert.Contains(t, body, "🔄 5 раундов")
	})

	t.Run("zero rounds is shown, not treated as absent", func(t *testing.T) {
		t.Parallel()

		var rounds int16

		body := info.Body(&model.Training{
			Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 60, Rounds: &rounds,
		})

		assert.Contains(t, body, "🔄 0 раундов")
	})

	t.Run("without rounds", func(t *testing.T) {
		t.Parallel()

		body := info.Body(&model.Training{Date: date, TrainingType: model.TrainingTypeGi, DurationMinutes: 60})

		assert.NotContains(t, body, "🔄")
	})
}

func TestCard(t *testing.T) {
	t.Parallel()

	card := info.Card(&model.Training{
		Date: time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC), TrainingType: model.TrainingTypeNoGi, DurationMinutes: 90,
	})

	assert.Contains(t, card, "Тренировка")
	assert.Contains(t, card, "30 сентября 2026")
}

func TestKeyboard(t *testing.T) {
	t.Parallel()

	kb := info.Keyboard(7)

	assert.Equal(t, "✏️ Изменить", kb[0][0].Label)
	assert.Equal(t, "training:edit:7", kb[0][0].Data)
	assert.Equal(t, "🗑 Удалить", kb[1][0].Label)
	assert.Equal(t, "training:delete:7", kb[1][0].Data)
	assert.Equal(t, "← Назад", kb[2][0].Label)
	assert.Equal(t, "training:history:page:0", kb[2][0].Data)
}

func TestTrainingTypeLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   model.TrainingType
		want string
	}{
		{name: "gi", in: model.TrainingTypeGi, want: "🥋 Gi"},
		{name: "no-gi", in: model.TrainingTypeNoGi, want: "🥷 No-Gi"},
		{name: "open mat", in: model.TrainingTypeOpenMat, want: "🤼 Open Mat"},
		{name: "unknown falls back to the raw value", in: model.TrainingType("bogus"), want: "bogus"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, info.TrainingTypeLabel(tc.in))
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		minutes int32
		want    string
	}{
		{minutes: 1, want: "1 минута"},
		{minutes: 2, want: "2 минуты"},
		{minutes: 4, want: "4 минуты"},
		{minutes: 5, want: "5 минут"},
		{minutes: 11, want: "11 минут"},
		{minutes: 21, want: "21 минута"},
		{minutes: 90, want: "90 минут"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, info.FormatDuration(tc.minutes))
	}
}

func TestFormatRounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		rounds int16
		want   string
	}{
		{rounds: 0, want: "0 раундов"},
		{rounds: 1, want: "1 раунд"},
		{rounds: 2, want: "2 раунда"},
		{rounds: 4, want: "4 раунда"},
		{rounds: 5, want: "5 раундов"},
		{rounds: 11, want: "11 раундов"},
		{rounds: 21, want: "21 раунд"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, info.FormatRounds(tc.rounds))
	}
}

func TestFormatDateShort(t *testing.T) {
	t.Parallel()

	got := info.FormatDateShort(time.Date(2026, 10, 7, 0, 0, 0, 0, time.UTC))

	assert.Equal(t, "7 октября", got)
}
