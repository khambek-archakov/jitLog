package info

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/model"
)

func TestBody(t *testing.T) {
	t.Parallel()

	text := Body(&model.ScheduleSlot{DayOfWeek: 3, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeNoGi})

	assert.Equal(t, "Ср — 19:00, 🥷 No-Gi", text)
}

func TestCard(t *testing.T) {
	t.Parallel()

	text := Card(&model.ScheduleSlot{DayOfWeek: 1, TimeMinutes: 11 * 60, TrainingType: model.TrainingTypeGi})

	assert.Contains(t, text, "Слот расписания")
	assert.Contains(t, text, "Пн — 11:00, 🥋 Gi")
}

func TestKeyboard(t *testing.T) {
	t.Parallel()

	kb := Keyboard(5)

	require.Len(t, kb, 3)
	assert.Equal(t, "schedule:edit:5", kb[0][0].Data)
	assert.Equal(t, "schedule:delete:5", kb[1][0].Data)
	assert.Equal(t, "schedule:list", kb[2][0].Data)
}

func TestDayLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Пн", DayLabel(1))
	assert.Equal(t, "Вс", DayLabel(7))
	assert.Equal(t, "—", DayLabel(0))
	assert.Equal(t, "—", DayLabel(8))
}

func TestFormatTime(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "19:00", FormatTime(19*60))
	assert.Equal(t, "09:05", FormatTime(9*60+5))
}

func TestTrainingTypeLabel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "🥋 Gi", TrainingTypeLabel(model.TrainingTypeGi))
	assert.Equal(t, "🥷 No-Gi", TrainingTypeLabel(model.TrainingTypeNoGi))
	assert.Equal(t, "🤼 Open Mat", TrainingTypeLabel(model.TrainingTypeOpenMat))
}
