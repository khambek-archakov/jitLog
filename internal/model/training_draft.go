package model

import "time"

type TrainingDraftStep string

const (
	TrainingDraftStepAwaitingDate     TrainingDraftStep = "awaiting_date"
	TrainingDraftStepAwaitingType     TrainingDraftStep = "awaiting_type"
	TrainingDraftStepAwaitingDuration TrainingDraftStep = "awaiting_duration"
	TrainingDraftStepAwaitingNotes    TrainingDraftStep = "awaiting_notes"
)

type TrainingDraft struct {
	ID              int64
	UserID          int64
	Step            TrainingDraftStep
	Date            *time.Time
	TrainingType    TrainingType
	DurationMinutes *int32
	Notes           *string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
