package model

import "time"

type TrainingType string

const (
	TrainingTypeNone    TrainingType = ""
	TrainingTypeGi      TrainingType = "Gi"
	TrainingTypeNoGi    TrainingType = "No-Gi"
	TrainingTypeOpenMat TrainingType = "Open Mat"
)

type Training struct {
	ID              int64
	UserID          int64
	Date            time.Time
	TrainingType    TrainingType
	DurationMinutes int32
	Notes           *string
	CreatedAt       time.Time
}
