package list

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/model"
)

func TestListText(t *testing.T) {
	t.Parallel()

	t.Run("empty state", func(t *testing.T) {
		t.Parallel()

		text := listText(nil)

		assert.Contains(t, text, "Пока нет ни одного слота")
	})

	t.Run("non-empty just shows the header — rows are buttons", func(t *testing.T) {
		t.Parallel()

		text := listText([]*model.ScheduleSlot{
			{DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi},
		})

		assert.Equal(t, "📅 Расписание", text)
	})
}

func TestListKeyboard(t *testing.T) {
	t.Parallel()

	t.Run("one row per slot plus Add and Back", func(t *testing.T) {
		t.Parallel()

		kb := listKeyboard([]*model.ScheduleSlot{
			{ID: 5, DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi},
			{ID: 7, DayOfWeek: 6, TimeMinutes: 11 * 60, TrainingType: model.TrainingTypeOpenMat},
		})

		require.Len(t, kb, 4)
		assert.Equal(t, "schedule:view:5", kb[0][0].Data)
		assert.Contains(t, kb[0][0].Label, "Пн")
		assert.Equal(t, "schedule:view:7", kb[1][0].Data)
		assert.Contains(t, kb[1][0].Label, "Сб")
		assert.Equal(t, "schedule:add", kb[2][0].Data)
		assert.Equal(t, "menu:back", kb[3][0].Data)
	})

	t.Run("empty state keyboard is just Add and Back", func(t *testing.T) {
		t.Parallel()

		kb := listKeyboard(nil)

		require.Len(t, kb, 2)
		assert.Equal(t, "schedule:add", kb[0][0].Data)
		assert.Equal(t, "menu:back", kb[1][0].Data)
	})
}
