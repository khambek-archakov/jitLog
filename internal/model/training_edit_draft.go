package model

import "time"

// TrainingEditField names which field of an existing training a pending
// free-text reply is meant for — only fields whose input can't be read off
// a callback button (duration's "Другое", and notes) need this.
type TrainingEditField string

const (
	TrainingEditFieldDuration TrainingEditField = "duration"
	TrainingEditFieldNotes    TrainingEditField = "notes"
)

// TrainingEditDraft is the minimal state kept while a user is mid-edit of
// an existing training and the next message from them is free text: which
// training, and which field it's for. Every other edit action (type,
// duration's quick buttons, date) rides entirely on callback data and
// never needs a row here.
type TrainingEditDraft struct {
	UserID     int64
	TrainingID int64
	Field      TrainingEditField
	CreatedAt  time.Time
}
