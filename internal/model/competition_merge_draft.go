package model

import "time"

// CompetitionMergeDraft is the state kept while the admin is searching for
// which existing Competition a pending submission (flagged "🔗 Это дубль")
// should be merged into — the only free-text step in the whole
// catalog/moderation flow. UserID here is the admin's own user row, not
// the original submitter's.
type CompetitionMergeDraft struct {
	UserID               int64
	PendingCompetitionID int64
	CreatedAt            time.Time
}
