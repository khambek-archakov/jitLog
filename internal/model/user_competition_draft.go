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
	ID     int64
	UserID int64
	Step   UserCompetitionDraftStep
	// FromCatalog remembers whether this wizard was started from "🔎 Найти
	// соревнование" (catalog's own "➕ Добавить" button) rather than
	// "🏆 Соревнования" — Отмена needs to know which screen to return to.
	// Packed into the same DB column as Step (see the repository's own
	// draftStepToDB/FromDB) rather than a column of its own.
	FromCatalog bool
	Title       *string
	Date        *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
