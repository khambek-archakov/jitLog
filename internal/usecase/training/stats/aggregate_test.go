package stats

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/khambek-archakov/jitLog/internal/model"
)

func trainingOn(dateStr string, trainingType model.TrainingType, minutes int32) *model.Training {
	d, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		panic(err)
	}

	return &model.Training{Date: d, TrainingType: trainingType, DurationMinutes: minutes}
}

func TestFilterByPeriod(t *testing.T) {
	t.Parallel()

	// Wednesday 2026-03-18 — week is 2026-03-16 (Mon) .. 2026-03-22 (Sun).
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	trainings := []*model.Training{
		trainingOn("2026-01-05", model.TrainingTypeGi, 60),   // earlier this year, outside week/month
		trainingOn("2026-03-01", model.TrainingTypeGi, 60),   // this month, outside week
		trainingOn("2026-03-16", model.TrainingTypeGi, 60),   // this week (Monday)
		trainingOn("2026-03-18", model.TrainingTypeNoGi, 90), // this week (today)
	}

	tests := []struct {
		name string
		p    period
		want int
	}{
		{name: "week keeps only this week's two", p: periodWeek, want: 2},
		{name: "month keeps the three in March", p: periodMonth, want: 3},
		{name: "year keeps all four", p: periodYear, want: 4},
		{name: "all keeps all four", p: periodAll, want: 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := filterByPeriod(trainings, tc.p, now)

			assert.Len(t, got, tc.want)
		})
	}
}

// TestFilterByPeriod_NowInNonUTCLocation guards the same regression as
// TestPeriodStart_IgnoresNowsLocation, but end-to-end: a training logged
// "today" must stay in the "month"/"week" period even when now carries a
// timezone behind UTC.
func TestFilterByPeriod_NowInNonUTCLocation(t *testing.T) {
	t.Parallel()

	behindUTC := time.FixedZone("UTC-8", -8*3600)
	now := time.Date(2026, 3, 18, 7, 0, 0, 0, behindUTC) // wall-clock 2026-03-18

	today := trainingOn("2026-03-18", model.TrainingTypeGi, 60)

	for _, p := range []period{periodWeek, periodMonth, periodYear} {
		got := filterByPeriod([]*model.Training{today}, p, now)
		assert.Len(t, got, 1, "period %s should still include a training logged today", p)
	}
}

func TestCountStats(t *testing.T) {
	t.Parallel()

	c := countStats([]*model.Training{
		trainingOn("2026-03-16", model.TrainingTypeGi, 60),
		trainingOn("2026-03-17", model.TrainingTypeGi, 90),
		trainingOn("2026-03-18", model.TrainingTypeNoGi, 45),
		trainingOn("2026-03-19", model.TrainingTypeOpenMat, 120),
	})

	assert.Equal(t, 4, c.Total)
	assert.Equal(t, int32(315), c.TotalMinutes)
	assert.Equal(t, 2, c.GiCount)
	assert.Equal(t, 1, c.NoGiCount)
	assert.Equal(t, 1, c.OpenMatCount)
}

func TestCounts_Hours(t *testing.T) {
	t.Parallel()

	tests := []struct {
		minutes int32
		want    int
	}{
		{minutes: 0, want: 0},
		{minutes: 59, want: 1},
		{minutes: 89, want: 1},
		{minutes: 91, want: 2},
		{minutes: 360, want: 6},
	}

	for _, tc := range tests {
		c := counts{TotalMinutes: tc.minutes}
		assert.Equal(t, tc.want, c.hours())
	}
}

func TestCounts_Percent(t *testing.T) {
	t.Parallel()

	c := counts{Total: 4}

	assert.Equal(t, 50, c.percent(2))
	assert.Equal(t, 25, c.percent(1))
	assert.Equal(t, 0, c.percent(0))
	assert.Equal(t, 0, counts{}.percent(0))
}

func TestStreak(t *testing.T) {
	t.Parallel()

	// Wednesday 2026-03-18 — this week starts Monday 2026-03-16.
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)

	t.Run("no trainings at all", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, 0, streak(nil, now))
	})

	t.Run("trained this week and the two before — streak of 3", func(t *testing.T) {
		t.Parallel()

		trainings := []*model.Training{
			trainingOn("2026-03-02", model.TrainingTypeGi, 60), // week of 2026-03-02
			trainingOn("2026-03-10", model.TrainingTypeGi, 60), // week of 2026-03-09
			trainingOn("2026-03-17", model.TrainingTypeGi, 60), // week of 2026-03-16 (this week)
		}

		assert.Equal(t, 3, streak(trainings, now))
	})

	t.Run("no training yet this week doesn't break a streak built on prior weeks", func(t *testing.T) {
		t.Parallel()

		trainings := []*model.Training{
			trainingOn("2026-03-10", model.TrainingTypeGi, 60), // week of 2026-03-09
			trainingOn("2026-03-12", model.TrainingTypeGi, 60), // same week
		}

		assert.Equal(t, 1, streak(trainings, now))
	})

	t.Run("a gap of a full empty week breaks the streak", func(t *testing.T) {
		t.Parallel()

		trainings := []*model.Training{
			trainingOn("2026-02-02", model.TrainingTypeGi, 60), // week of 2026-02-02, long gone
			trainingOn("2026-03-17", model.TrainingTypeGi, 60), // this week
		}

		assert.Equal(t, 1, streak(trainings, now))
	})
}
