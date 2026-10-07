package model

import "time"

// CompetitionStatus is the catalog moderation state — never set by a
// personal user_competition row, only by the admin via catalog/moderate.
type CompetitionStatus string

const (
	CompetitionStatusPending   CompetitionStatus = "pending"
	CompetitionStatusPublished CompetitionStatus = "published"
	CompetitionStatusRejected  CompetitionStatus = "rejected"
)

// Competition is a shared catalog entry — the thing many users' own
// UserCompetition rows can link to via CompetitionID once a submission is
// approved (or auto-attached via an exact URL/candidate match). Unlike
// UserCompetition, nothing here is user-specific.
type Competition struct {
	ID        int64
	Title     string
	Date      time.Time
	EndDate   *time.Time
	City      *string
	Status    CompetitionStatus
	CreatedBy *int64
	CreatedAt time.Time
}

// CompetitionSource is one URL known to point at a Competition — a
// tournament can have several (the organizer's own page, an aggregator's
// listing, ...). Exactly one per Competition is marked primary (enforced
// by a partial unique index on competition_id where is_primary), and only
// the admin ever picks which one that is.
type CompetitionSource struct {
	CompetitionID int64
	URL           string
	IsPrimary     bool
}
