package model

import "time"

// CompetitionImportDiff is a pending "the source says this competition's
// date/city changed" proposal from an automated import (currently just
// AJP) — the import never overwrites the stored record itself, it leaves
// one of these for the admin to accept ("Обновить") or dismiss
// ("Оставить") via competition/moderate.
type CompetitionImportDiff struct {
	CompetitionID   int64
	ProposedDate    time.Time
	ProposedEndDate *time.Time
	ProposedCity    *string
	CreatedAt       time.Time
}
