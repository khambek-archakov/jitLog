package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/model"
)

func TestBar(t *testing.T) {
	t.Parallel()

	tests := []struct {
		percent int
		want    string
	}{
		{percent: 0, want: "░░░░░░░░░░"},
		{percent: 50, want: "█████░░░░░"},
		{percent: 100, want: "██████████"},
		{percent: 25, want: "███░░░░░░░"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, bar(tc.percent))
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		minutes int32
		want    string
	}{
		{minutes: 0, want: "0 мин"},
		{minutes: 45, want: "45 мин"},
		{minutes: 60, want: "1 ч"},
		{minutes: 180, want: "3 ч"},
		{minutes: 165, want: "2 ч 45 мин"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, formatDuration(tc.minutes))
	}
}

func TestWeeksWord(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int
		want string
	}{
		{n: 1, want: "неделя"},
		{n: 2, want: "недели"},
		{n: 4, want: "недели"},
		{n: 5, want: "недель"},
		{n: 11, want: "недель"},
		{n: 21, want: "неделя"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, weeksWord(tc.n))
	}
}

func TestStreakLine(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "", streakLine(0))
	assert.Equal(t, "🔥 Серия — 3 недели подряд", streakLine(3))
}

func TestStatsText(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	t.Run("with data shows counts, hours and type breakdown", func(t *testing.T) {
		t.Parallel()

		c := counts{Total: 4, TotalMinutes: 360, GiCount: 2, NoGiCount: 1, OpenMatCount: 1}

		text := statsText(periodWeek, now, c, 3)

		assert.Contains(t, text, "Неделя")
		assert.Contains(t, text, "16–22 марта")
		assert.Contains(t, text, "🥋 Тренировок: 4")
		assert.Contains(t, text, "⏱ Время на мате: 6 ч")
		assert.Contains(t, text, "🥋 Gi")
		assert.Contains(t, text, "50%")
		assert.Contains(t, text, "🔥 Серия — 3 недели подряд")
	})

	t.Run("empty period still shows the streak", func(t *testing.T) {
		t.Parallel()

		text := statsText(periodMonth, now, counts{}, 2)

		assert.Contains(t, text, "В этот период тренировок не было.")
		assert.Contains(t, text, "🔥 Серия — 2 недели подряд")
	})

	t.Run("no streak omits the streak line entirely", func(t *testing.T) {
		t.Parallel()

		text := statsText(periodAll, now, counts{Total: 1, TotalMinutes: 60, GiCount: 1}, 0)

		assert.NotContains(t, text, "Серия")
	})
}

func TestStatsKeyboard(t *testing.T) {
	t.Parallel()

	t.Run("without the mode switch", func(t *testing.T) {
		t.Parallel()

		kb := statsKeyboard(periodMonth, false)

		require.Len(t, kb, 2)
		require.Len(t, kb[0], 4)

		assert.Equal(t, "Неделя", kb[0][0].Label)
		assert.Equal(t, "stats:period:week", kb[0][0].Data)

		assert.Equal(t, "• Месяц •", kb[0][1].Label)
		assert.Equal(t, "stats:period:month", kb[0][1].Data)

		assert.Equal(t, "Год", kb[0][2].Label)
		assert.Equal(t, "Всё время", kb[0][3].Label)

		assert.Equal(t, "← Главное меню", kb[1][0].Label)
		assert.Equal(t, "menu:back", kb[1][0].Data)
	})

	t.Run("with the mode switch", func(t *testing.T) {
		t.Parallel()

		kb := statsKeyboard(periodWeek, true)

		require.Len(t, kb, 3)
		assert.Equal(t, "• За период •", kb[1][0].Label)
		assert.Equal(t, "stats:period:week", kb[1][0].Data)
		assert.Equal(t, "По поясам", kb[1][1].Label)
		assert.Equal(t, "stats:belts", kb[1][1].Data)

		assert.Equal(t, "← Главное меню", kb[2][0].Label)
	})
}

func TestBeltsKeyboard(t *testing.T) {
	t.Parallel()

	kb := beltsKeyboard()

	require.Len(t, kb, 2)
	assert.Equal(t, "За период", kb[0][0].Label)
	assert.Equal(t, "stats:period:week", kb[0][0].Data)
	assert.Equal(t, "• По поясам •", kb[0][1].Label)
	assert.Equal(t, "stats:belts", kb[0][1].Data)

	assert.Equal(t, "← Главное меню", kb[1][0].Label)
}

func TestBeltsText(t *testing.T) {
	t.Parallel()

	promotions := []*model.BeltPromotion{
		{Belt: model.BeltWhite, PromotedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Belt: model.BeltBlue, PromotedAt: time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)},
	}

	t.Run("with data shows the breakdown in belt order", func(t *testing.T) {
		t.Parallel()

		c := beltCounts{Total: 4, TotalMinutes: 240, perBelt: map[model.Belt]int{
			model.BeltWhite: 1,
			model.BeltBlue:  3,
		}}

		text := beltsText(c, promotions, 2)

		assert.Contains(t, text, "По поясам")
		assert.Contains(t, text, "🥋 Тренировок: 4")
		assert.Contains(t, text, "⏱ Время на мате: 4 ч")
		assert.Contains(t, text, "⚪ Белый")
		assert.Contains(t, text, "25%")
		assert.Contains(t, text, "🔵 Синий")
		assert.Contains(t, text, "75%")
		assert.Contains(t, text, "🔥 Серия — 2 недели подряд")

		// Belts never held shouldn't show up at all.
		assert.NotContains(t, text, "Пурпурный")
	})

	t.Run("no trainings yet", func(t *testing.T) {
		t.Parallel()

		text := beltsText(beltCounts{perBelt: map[model.Belt]int{}}, promotions, 0)

		assert.Contains(t, text, "Тренировок пока нет.")
	})
}

func TestEmptyState(t *testing.T) {
	t.Parallel()

	assert.Contains(t, emptyStateText(), "Пока нет ни одной тренировки")

	kb := emptyStateKeyboard()

	require.Len(t, kb, 2)
	assert.Equal(t, "menu:add_training", kb[0][0].Data)
	assert.Equal(t, "menu:back", kb[1][0].Data)
}
