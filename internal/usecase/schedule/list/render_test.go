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

		text := listText(nil, &model.User{})

		assert.Contains(t, text, "Пока нет ни одного слота")
	})

	t.Run("non-empty shows the reminder status — off by default", func(t *testing.T) {
		t.Parallel()

		text := listText([]*model.ScheduleSlot{
			{DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi},
		}, &model.User{})

		assert.Contains(t, text, "📅 Расписание")
		assert.Contains(t, text, "Напоминания выключены")
	})

	t.Run("non-empty shows the zone label once reminders are on", func(t *testing.T) {
		t.Parallel()

		tz := "Europe/Samara"
		text := listText([]*model.ScheduleSlot{
			{DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi},
		}, &model.User{Timezone: &tz, RemindersEnabled: true})

		assert.Contains(t, text, "Напоминания включены. Часовой пояс: Самара, Баку, Ереван +4")
	})
}

func TestListKeyboard(t *testing.T) {
	t.Parallel()

	t.Run("one row per slot plus Add, toggle and Back", func(t *testing.T) {
		t.Parallel()

		kb := listKeyboard([]*model.ScheduleSlot{
			{ID: 5, DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi},
			{ID: 7, DayOfWeek: 6, TimeMinutes: 11 * 60, TrainingType: model.TrainingTypeOpenMat},
		}, &model.User{})

		require.Len(t, kb, 5)
		assert.Equal(t, "schedule:view:5", kb[0][0].Data)
		assert.Contains(t, kb[0][0].Label, "Пн")
		assert.Equal(t, "schedule:view:7", kb[1][0].Data)
		assert.Contains(t, kb[1][0].Label, "Сб")
		assert.Equal(t, "schedule:add", kb[2][0].Data)
		assert.Equal(t, "schedule:reminders:toggle", kb[3][0].Data)
		assert.Equal(t, "🔔 Напоминание: выкл", kb[3][0].Label)
		assert.Equal(t, "menu:back", kb[4][0].Data)
	})

	t.Run("reminders on shows the вкл label", func(t *testing.T) {
		t.Parallel()

		kb := listKeyboard([]*model.ScheduleSlot{
			{ID: 5, DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi},
		}, &model.User{RemindersEnabled: true})

		assert.Equal(t, "🔔 Напоминание: вкл", kb[2][0].Label)
	})

	t.Run("empty state keyboard is just Add and Back, no toggle", func(t *testing.T) {
		t.Parallel()

		kb := listKeyboard(nil, &model.User{})

		require.Len(t, kb, 2)
		assert.Equal(t, "schedule:add", kb[0][0].Data)
		assert.Equal(t, "menu:back", kb[1][0].Data)
	})
}

func TestTZPickerKeyboard(t *testing.T) {
	t.Parallel()

	kb := tzPickerKeyboard()

	require.Len(t, kb, 4)
	assert.Equal(t, "schedule:tz:set:msk", kb[0][0].Data)
	assert.Equal(t, "schedule:tz:set:sam", kb[1][0].Data)
	assert.Equal(t, "schedule:tz:set:yek", kb[2][0].Data)
	assert.Equal(t, "schedule:tz:cancel", kb[3][0].Data)
}
