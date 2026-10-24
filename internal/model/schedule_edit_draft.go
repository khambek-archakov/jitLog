package model

import "time"

// ScheduleEditField names which field of an existing schedule slot a
// pending free-text reply is meant for — only fields whose input can't be
// read off a callback button (time's and duration's own "Другое") need
// this.
type ScheduleEditField string

const (
	ScheduleEditFieldTime     ScheduleEditField = "time"
	ScheduleEditFieldDuration ScheduleEditField = "duration"
)

// ScheduleEditDraft is the minimal state kept while a user is mid-edit of
// an existing schedule slot and the next message from them is free text:
// which slot, and which field it's for. Day and training type are always
// picked from buttons and never need a row here.
type ScheduleEditDraft struct {
	UserID    int64
	SlotID    int64
	Field     ScheduleEditField
	CreatedAt time.Time
}
