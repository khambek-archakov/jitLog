package model

import "time"

// ScheduleEditDraft is the minimal state kept while a user is mid-edit of
// an existing schedule slot and the next message from them is free text.
// Unlike TrainingEditDraft, there's no Field to track — day and training
// type are always picked from buttons, so time (via its "Другое" fallback)
// is the only field a free-text reply can ever be for.
type ScheduleEditDraft struct {
	UserID    int64
	SlotID    int64
	CreatedAt time.Time
}
