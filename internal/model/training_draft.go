package model

import "time"

type TrainingDraftStep string

const (
	TrainingDraftStepAwaitingDate     TrainingDraftStep = "awaiting_date"
	TrainingDraftStepAwaitingType     TrainingDraftStep = "awaiting_type"
	TrainingDraftStepAwaitingDuration TrainingDraftStep = "awaiting_duration"
)

// TrainingDraft is deliberately just three questions (date, type,
// duration) — rounds and notes are added after the training is already
// saved, via the edit flow, to keep the create wizard as short as
// possible.
type TrainingDraft struct {
	ID              int64
	UserID          int64
	Step            TrainingDraftStep
	Date            *time.Time
	TrainingType    TrainingType
	DurationMinutes *int32
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
