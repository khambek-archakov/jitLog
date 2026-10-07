package info_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
)

func date(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func TestStatus(t *testing.T) {
	t.Parallel()

	today := date(2026, 11, 1)

	tests := []struct {
		name string
		c    *model.UserCompetition
		want string
	}{
		{
			name: "tomorrow",
			c:    &model.UserCompetition{Date: date(2026, 11, 2)},
			want: "Завтра",
		},
		{
			name: "in N days",
			c:    &model.UserCompetition{Date: date(2026, 11, 24)},
			want: "Через 23 дня",
		},
		{
			name: "ongoing single day",
			c:    &model.UserCompetition{Date: date(2026, 11, 1)},
			want: "Идёт сейчас",
		},
		{
			name: "ongoing multi-day, today inside the range",
			c:    &model.UserCompetition{Date: date(2026, 10, 30), EndDate: ptr(date(2026, 11, 3))},
			want: "Идёт сейчас",
		},
		{
			name: "ended 5 days ago",
			c:    &model.UserCompetition{Date: date(2026, 10, 27)},
			want: "Прошёл 5 дней назад",
		},
		{
			name: "multi-day ended, measured from end_date not start",
			c:    &model.UserCompetition{Date: date(2026, 10, 20), EndDate: ptr(date(2026, 10, 30))},
			want: "Прошёл 2 дня назад",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, info.Status(tc.c, today))
		})
	}
}

func TestBody(t *testing.T) {
	t.Parallel()

	today := date(2026, 11, 1)

	t.Run("full card before the tournament — no result line even if set", func(t *testing.T) {
		t.Parallel()

		result := "should not show yet"
		body := info.Body(&model.UserCompetition{
			Title: "Moscow Open", Date: date(2026, 11, 24), City: ptr("Москва"), Result: &result,
		}, today)

		assert.Contains(t, body, "Moscow Open")
		assert.Contains(t, body, "24 ноября 2026")
		assert.Contains(t, body, "🏙 Москва")
		assert.Contains(t, body, "Через 23 дня")
		assert.NotContains(t, body, "🏅")
	})

	t.Run("result line appears once the tournament has started", func(t *testing.T) {
		t.Parallel()

		result := "2nd place"
		body := info.Body(&model.UserCompetition{
			Title: "Moscow Open", Date: date(2026, 10, 20), Result: &result,
		}, today)

		assert.Contains(t, body, "🏅 2nd place")
	})

	t.Run("date range when end_date is set", func(t *testing.T) {
		t.Parallel()

		body := info.Body(&model.UserCompetition{
			Title: "Worlds", Date: date(2026, 11, 20), EndDate: ptr(date(2026, 11, 22)),
		}, today)

		assert.Contains(t, body, "20 ноября 2026 – 22 ноября 2026")
	})

	t.Run("no city — no city line", func(t *testing.T) {
		t.Parallel()

		body := info.Body(&model.UserCompetition{Title: "Worlds", Date: date(2026, 11, 20)}, today)

		assert.NotContains(t, body, "🏙")
	})
}

func TestKeyboard(t *testing.T) {
	t.Parallel()

	today := date(2026, 11, 1)

	t.Run("no link, before start, nothing filled in — add toggles, no result row", func(t *testing.T) {
		t.Parallel()

		kb := info.Keyboard(&model.UserCompetition{ID: 7, Date: date(2026, 11, 24)}, today)

		assert.Equal(t, "➕ Город", kb[0][0].Label)
		assert.Equal(t, "competition:edit:7:city", kb[0][0].Data)
		assert.Equal(t, "➕ Ссылка", kb[0][1].Label)
		assert.Equal(t, "competition:edit:7:url", kb[0][1].Data)
		assert.Equal(t, "➕ Дата окончания", kb[1][0].Label)
		assert.Equal(t, "competition:edit:7:end_date", kb[1][0].Data)
		assert.Equal(t, "✏️ Изменить", kb[2][0].Label)
		assert.Equal(t, "competition:edit:7", kb[2][0].Data)
		assert.Equal(t, "🗑️ Удалить", kb[3][0].Label)
		assert.Equal(t, "competition:delete:7", kb[3][0].Data)
		assert.Equal(t, "← Назад", kb[4][0].Label)
		assert.Equal(t, "competition:list", kb[4][0].Data)
	})

	t.Run("filled fields toggle to edit icons", func(t *testing.T) {
		t.Parallel()

		city, url := "Москва", "https://example.com"
		kb := info.Keyboard(&model.UserCompetition{
			ID: 7, Date: date(2026, 11, 24), City: &city, URL: &url, EndDate: ptr(date(2026, 11, 25)),
		}, today)

		// URL button comes first since a link is set.
		assert.Equal(t, "✏️ Город", kb[1][0].Label)
		assert.Equal(t, "✏️ Ссылка", kb[1][1].Label)
		assert.Equal(t, "✏️ Дата окончания", kb[2][0].Label)
	})

	t.Run("link present — URL button first", func(t *testing.T) {
		t.Parallel()

		url := "https://example.com"
		kb := info.Keyboard(&model.UserCompetition{ID: 7, Date: date(2026, 11, 24), URL: &url}, today)

		assert.Equal(t, "🔗 Страница турнира", kb[0][0].Label)
		assert.Equal(t, url, kb[0][0].URL)
	})

	t.Run("started — result shortcut appears", func(t *testing.T) {
		t.Parallel()

		kb := info.Keyboard(&model.UserCompetition{ID: 7, Date: date(2026, 10, 20)}, today)

		assert.Equal(t, "🏅 Результат", kb[2][0].Label)
		assert.Equal(t, "competition:edit:7:result", kb[2][0].Data)
	})
}

func TestFormatDateRow(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "15 ноя", info.FormatDateRow(date(2026, 11, 15)))
	assert.Equal(t, "1 янв", info.FormatDateRow(date(2026, 1, 1)))
}

func TestFormatDays(t *testing.T) {
	t.Parallel()

	tests := []struct {
		n    int
		want string
	}{
		{n: 1, want: "1 день"},
		{n: 2, want: "2 дня"},
		{n: 4, want: "4 дня"},
		{n: 5, want: "5 дней"},
		{n: 11, want: "11 дней"},
		{n: 21, want: "21 день"},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.want, info.FormatDays(tc.n))
	}
}

func TestToday(t *testing.T) {
	t.Parallel()

	t.Run("no timezone falls back to UTC", func(t *testing.T) {
		t.Parallel()

		got := info.Today(&model.User{})
		want := time.Now().UTC()

		assert.Equal(t, time.Date(want.Year(), want.Month(), want.Day(), 0, 0, 0, 0, time.UTC), got)
	})

	t.Run("invalid timezone falls back to UTC instead of erroring", func(t *testing.T) {
		t.Parallel()

		tz := "not/a-real-zone"

		assert.NotPanics(t, func() {
			info.Today(&model.User{Timezone: &tz})
		})
	})
}

func ptr[T any](v T) *T {
	return &v
}
