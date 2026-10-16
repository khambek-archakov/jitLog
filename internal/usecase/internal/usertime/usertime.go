// Package usertime centralizes "what time is it for this user" — every
// scenario that used to call time.Now() directly for a date/week/month
// boundary (competition countdowns, stats, the reminder job) goes through
// here instead, so they all agree on the same per-user local clock.
package usertime

import (
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
)

// DefaultTimezone is what every boundary falls back to when a user has no
// Timezone set yet — Moscow, not UTC, per the reminder feature's own spec
// (most users are in or near it, so it's a better blind guess than UTC).
const DefaultTimezone = "Europe/Moscow"

// Location resolves u's own IANA zone, falling back to DefaultTimezone when
// Timezone is unset or isn't a name time.LoadLocation recognizes.
func Location(u *model.User) *time.Location {
	if u.Timezone != nil {
		if l, err := time.LoadLocation(*u.Timezone); err == nil {
			return l
		}
	}

	loc, err := time.LoadLocation(DefaultTimezone)
	if err != nil {
		return time.UTC
	}

	return loc
}

// Now is the current instant, in u's own Location.
func Now(u *model.User) time.Time {
	return time.Now().In(Location(u))
}

// Today is "today" as a UTC-midnight time.Time (matching how a date column
// round-trips through pgx), using u's own Location to decide which
// calendar day "today" actually is.
func Today(u *model.User) time.Time {
	now := Now(u)

	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// StartOfWeek returns the Monday (00:00 UTC) of now's own calendar week,
// using now's own (local) year/month/day — a pure function of now, so
// callers needing a fake clock for tests (the reminder job) just build now
// in the zone/instant they want to test, no mockable clock of its own
// needed here.
func StartOfWeek(now time.Time) time.Time {
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // time.Sunday == 0 — shift so Monday=1..Sunday=7.
	}

	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	return day.AddDate(0, 0, -(weekday - 1))
}
