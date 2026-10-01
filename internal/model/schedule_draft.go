package model

import "time"

type ScheduleDraftStep string

const (
	ScheduleDraftStepAwaitingDay  ScheduleDraftStep = "awaiting_day"
	ScheduleDraftStepAwaitingTime ScheduleDraftStep = "awaiting_time"
	ScheduleDraftStepAwaitingType ScheduleDraftStep = "awaiting_type"
)

type ScheduleDraft struct {
	ID           int64
	UserID       int64
	Step         ScheduleDraftStep
	DayOfWeek    *int16
	TimeMinutes  *int16
	TrainingType TrainingType
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
