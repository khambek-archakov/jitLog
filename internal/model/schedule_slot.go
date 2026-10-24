package model

import "time"

// ScheduleSlot is a recurring weekly commitment — "every Monday at 19:00 I
// train Gi" — not tied to any logged Training. DayOfWeek is 1=Monday..
// 7=Sunday (ISO weekday numbering); TimeMinutes is minutes since midnight
// (0-1439), avoiding any dependency on Postgres's own time-of-day codec.
type ScheduleSlot struct {
	ID           int64
	UserID       int64
	DayOfWeek    int16
	TimeMinutes  int16
	TrainingType TrainingType
	// DurationMinutes defaults to 90 at creation (see schedule_slot's own
	// migration) and is only ever changed from the slot's own card — the
	// post-session reminder (internal/usecase/reminder) uses it to know
	// when the session actually ends.
	DurationMinutes int16
	CreatedAt       time.Time
}
