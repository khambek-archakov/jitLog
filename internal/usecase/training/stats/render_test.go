package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		assert.Contains(t, text, "Тренировок: 4")
		assert.Contains(t, text, "Часов на мате: 6")
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

	kb := statsKeyboard(periodMonth)

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
}

func TestEmptyState(t *testing.T) {
	t.Parallel()

	assert.Contains(t, emptyStateText(), "Пока нет ни одной тренировки")

	kb := emptyStateKeyboard()

	require.Len(t, kb, 2)
	assert.Equal(t, "menu:add_training", kb[0][0].Data)
	assert.Equal(t, "menu:back", kb[1][0].Data)
}
