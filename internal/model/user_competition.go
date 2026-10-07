package model

import "time"

// UserCompetition is a tournament entry in a user's own personal list.
// CompetitionID stays nil for every manually-entered record — it's only
// ever set once stage 2 (the shared catalog) exists and a record gets
// linked to a catalog entry.
type UserCompetition struct {
	ID            int64
	UserID        int64
	CompetitionID *int64
	Title         string
	Date          time.Time
	EndDate       *time.Time
	City          *string
	URL           *string
	Result        *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
