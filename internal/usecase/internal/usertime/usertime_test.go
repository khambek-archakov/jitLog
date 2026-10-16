package usertime_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/usertime"
)

func TestLocation(t *testing.T) {
	t.Parallel()

	t.Run("nil timezone falls back to Europe/Moscow", func(t *testing.T) {
		t.Parallel()

		loc := usertime.Location(&model.User{})

		moscow, err := time.LoadLocation("Europe/Moscow")
		require.NoError(t, err)
		assert.Equal(t, moscow, loc)
	})

	t.Run("invalid timezone falls back to Europe/Moscow", func(t *testing.T) {
		t.Parallel()

		tz := "not/a-real-zone"
		loc := usertime.Location(&model.User{Timezone: &tz})

		moscow, err := time.LoadLocation("Europe/Moscow")
		require.NoError(t, err)
		assert.Equal(t, moscow, loc)
	})

	t.Run("a known IANA name resolves to it", func(t *testing.T) {
		t.Parallel()

		tz := "Asia/Yekaterinburg"
		loc := usertime.Location(&model.User{Timezone: &tz})

		yek, err := time.LoadLocation("Asia/Yekaterinburg")
		require.NoError(t, err)
		assert.Equal(t, yek, loc)
	})
}

func TestStartOfWeek(t *testing.T) {
	t.Parallel()

	t.Run("same UTC instant yields different local calendar dates in different zones", func(t *testing.T) {
		t.Parallel()

		// 2026-10-19 is a Monday. At 00:30 UTC, Kaliningrad (+2) is already
		// well into Monday (02:30), but Yekaterinburg (+5) is at 05:30 —
		// still Monday too here, so pick an instant where they actually
		// disagree: just after Sunday 22:00 UTC, Kaliningrad has crossed
		// into Monday but Yekaterinburg (+5, i.e. +5h) has too — instead
		// use just before midnight UTC on a Sunday, where a +5 zone has
		// already crossed into Monday but a +2 zone hasn't yet.
		instant := time.Date(2026, 10, 18, 20, 30, 0, 0, time.UTC) // Sunday 20:30 UTC

		kgd, err := time.LoadLocation("Europe/Kaliningrad")
		require.NoError(t, err)
		yek, err := time.LoadLocation("Asia/Yekaterinburg")
		require.NoError(t, err)

		nowKgd := instant.In(kgd) // Sunday 22:30 +2 — still Sunday
		nowYek := instant.In(yek) // Monday 01:30 +5 — already Monday

		mondayFromKgd := usertime.StartOfWeek(nowKgd)
		mondayFromYek := usertime.StartOfWeek(nowYek)

		assert.NotEqual(t, mondayFromKgd, mondayFromYek)
		assert.Equal(t, time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), mondayFromKgd)
		assert.Equal(t, time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC), mondayFromYek)
	})

	t.Run("Monday itself is its own start of week", func(t *testing.T) {
		t.Parallel()

		monday := time.Date(2026, 10, 19, 10, 0, 0, 0, time.UTC)
		assert.Equal(t, time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC), usertime.StartOfWeek(monday))
	})

	t.Run("Sunday rolls back to the same week's Monday", func(t *testing.T) {
		t.Parallel()

		sunday := time.Date(2026, 10, 25, 23, 59, 0, 0, time.UTC)
		assert.Equal(t, time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC), usertime.StartOfWeek(sunday))
	})
}

func TestToday(t *testing.T) {
	t.Parallel()

	tz := "Asia/Yekaterinburg"
	got := usertime.Today(&model.User{Timezone: &tz})

	loc, err := time.LoadLocation(tz)
	require.NoError(t, err)

	want := time.Now().In(loc)
	assert.Equal(t, time.Date(want.Year(), want.Month(), want.Day(), 0, 0, 0, 0, time.UTC), got)
}
