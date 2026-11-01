package stats

import (
	"math"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
)

type counts struct {
	Total        int
	TotalMinutes int32
	TotalRounds  int32
	GiCount      int
	NoGiCount    int
	OpenMatCount int
}

// filterByPeriod returns the subset of trainings that falls within p's
// own [from, to] bounds, relative to now.
func filterByPeriod(trainings []*model.Training, p period, now time.Time) []*model.Training {
	from, to := periodRange(p, now)

	return filterByRange(trainings, from, to)
}

// filterByRange returns the subset of trainings whose Date falls within
// [from, to], both inclusive calendar days — the custom-range screen's
// own filter, and what filterByPeriod itself is built on. A zero from
// (periodAll's own lower bound) excludes nothing, since every real date
// is after it. The upper bound is checked against the start of the day
// after to, rather than to itself, so a Date carrying a nonzero
// time-of-day (as every *model.Training the repository returns legitimately
// might, despite the column itself being just a date) still counts as
// "on" to, not after it.
func filterByRange(trainings []*model.Training, from, to time.Time) []*model.Training {
	exclusiveEnd := to.AddDate(0, 0, 1)

	var out []*model.Training

	for _, t := range trainings {
		if t.Date.Before(from) || !t.Date.Before(exclusiveEnd) {
			continue
		}

		out = append(out, t)
	}

	return out
}

func countStats(trainings []*model.Training) counts {
	var c counts

	for _, t := range trainings {
		c.Total++
		c.TotalMinutes += t.DurationMinutes

		if t.Rounds != nil {
			c.TotalRounds += int32(*t.Rounds)
		}

		switch t.TrainingType {
		case model.TrainingTypeGi:
			c.GiCount++
		case model.TrainingTypeNoGi:
			c.NoGiCount++
		case model.TrainingTypeOpenMat:
			c.OpenMatCount++
		default:
		}
	}

	return c
}

func (c counts) percent(n int) int {
	if c.Total == 0 {
		return 0
	}

	return int(math.Round(float64(n) / float64(c.Total) * 100))
}

// beltCounts is countStats' shape, but broken down by belt instead of
// training type.
type beltCounts struct {
	Total        int
	TotalMinutes int32
	perBelt      map[model.Belt]int
}

func (c beltCounts) percent(b model.Belt) int {
	if c.Total == 0 {
		return 0
	}

	return int(math.Round(float64(c.perBelt[b]) / float64(c.Total) * 100))
}

// countByBelt groups every training by the belt that was active on its
// date (see asOfBelt). promotions must be sorted ascending by PromotedAt
// and non-empty — callers only reach here once that's already true (see
// UseCase.Handle).
func countByBelt(trainings []*model.Training, promotions []*model.BeltPromotion) beltCounts {
	c := beltCounts{perBelt: make(map[model.Belt]int, len(promotions))}

	for _, t := range trainings {
		c.Total++
		c.TotalMinutes += t.DurationMinutes
		c.perBelt[asOfBelt(promotions, t.Date)]++
	}

	return c
}

// asOfBelt returns the belt active on date, per promotions (sorted
// ascending by PromotedAt). A date earlier than every promotion clamps to
// the earliest known belt — see the discussion that led to this: a
// training logged before signup almost always predates the only belt we
// know about anyway, and "unknown" is a worse default than "probably this
// one."
func asOfBelt(promotions []*model.BeltPromotion, date time.Time) model.Belt {
	belt := promotions[0].Belt

	for _, p := range promotions {
		if p.PromotedAt.After(date) {
			break
		}

		belt = p.Belt
	}

	return belt
}

// streak is how many consecutive calendar weeks (Monday-start), counting
// back from now, have at least one training. A week still in progress with
// no training yet doesn't break the streak — it's just not counted until
// it's over or a training is logged in it.
func streak(trainings []*model.Training, now time.Time) int {
	weeks := make(map[time.Time]bool, len(trainings))
	for _, t := range trainings {
		weeks[mondayOf(t.Date)] = true
	}

	cursor := mondayOf(now)
	if !weeks[cursor] {
		cursor = cursor.AddDate(0, 0, -7)
	}

	n := 0
	for weeks[cursor] {
		n++
		cursor = cursor.AddDate(0, 0, -7)
	}

	return n
}
