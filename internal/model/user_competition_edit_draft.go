package model

import "time"

// UserCompetitionEditField names which field of an existing user_competition
// row a pending free-text reply is meant for. Unlike training/schedule,
// every editable field here is free text — nothing in this feature is
// pickable from a button — so this covers the full field set.
type UserCompetitionEditField string

const (
	UserCompetitionEditFieldTitle   UserCompetitionEditField = "title"
	UserCompetitionEditFieldDate    UserCompetitionEditField = "date"
	UserCompetitionEditFieldEndDate UserCompetitionEditField = "end_date"
	UserCompetitionEditFieldCity    UserCompetitionEditField = "city"
	UserCompetitionEditFieldURL     UserCompetitionEditField = "url"
	UserCompetitionEditFieldResult  UserCompetitionEditField = "result"
)

// UserCompetitionEditDraft is the minimal state kept while a user is
// mid-edit of one field of an existing tournament entry and the next
// message from them is free text for it.
type UserCompetitionEditDraft struct {
	UserID            int64
	UserCompetitionID int64
	Field             UserCompetitionEditField
	CreatedAt         time.Time
}
