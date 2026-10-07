package model

import "time"

type UserCompetitionDraftStep string

const (
	UserCompetitionDraftStepAwaitingTitle UserCompetitionDraftStep = "awaiting_title"
	UserCompetitionDraftStepAwaitingDate  UserCompetitionDraftStep = "awaiting_date"
)

// UserCompetitionDraft is deliberately just two questions (title, date) —
// every other field (city, link, end date, result) is added after the
// record is already saved, via the edit flow, to keep adding a tournament
// as fast as possible.
type UserCompetitionDraft struct {
	ID        int64
	UserID    int64
	Step      UserCompetitionDraftStep
	Title     *string
	Date      *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}
