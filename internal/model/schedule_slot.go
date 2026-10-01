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
	CreatedAt    time.Time
}
