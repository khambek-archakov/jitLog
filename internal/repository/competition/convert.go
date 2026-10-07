package competition

import "github.com/khambek-archakov/jitLog/internal/model"

// draftStepToDB/FromDB pack both Step and FromCatalog into the single
// user_competition_draft.step smallint column — FromCatalog adds 2 to
// whichever base value Step alone would've used, rather than needing a
// column of its own.
func draftStepToDB(s model.UserCompetitionDraftStep, fromCatalog bool) int16 {
	v := int16(0)
	if s == model.UserCompetitionDraftStepAwaitingDate {
		v = 1
	}

	if fromCatalog {
		v += 2
	}

	return v
}

func draftStepFromDB(v int16) (model.UserCompetitionDraftStep, bool) {
	fromCatalog := v >= 2
	if fromCatalog {
		v -= 2
	}

	if v == 1 {
		return model.UserCompetitionDraftStepAwaitingDate, fromCatalog
	}

	return model.UserCompetitionDraftStepAwaitingTitle, fromCatalog
}

func editFieldToDB(f model.UserCompetitionEditField) int16 {
	switch f {
	case model.UserCompetitionEditFieldTitle:
		return 1
	case model.UserCompetitionEditFieldDate:
		return 2
	case model.UserCompetitionEditFieldEndDate:
		return 3
	case model.UserCompetitionEditFieldCity:
		return 4
	case model.UserCompetitionEditFieldURL:
		return 5
	case model.UserCompetitionEditFieldResult:
		return 6
	default:
		return 0
	}
}

func editFieldFromDB(v int16) model.UserCompetitionEditField {
	switch v {
	case 1:
		return model.UserCompetitionEditFieldTitle
	case 2:
		return model.UserCompetitionEditFieldDate
	case 3:
		return model.UserCompetitionEditFieldEndDate
	case 4:
		return model.UserCompetitionEditFieldCity
	case 5:
		return model.UserCompetitionEditFieldURL
	case 6:
		return model.UserCompetitionEditFieldResult
	default:
		return ""
	}
}

func competitionStatusToDB(s model.CompetitionStatus) int16 {
	switch s {
	case model.CompetitionStatusPending:
		return 1
	case model.CompetitionStatusPublished:
		return 2
	case model.CompetitionStatusRejected:
		return 3
	default:
		return 0
	}
}

func competitionStatusFromDB(v int16) model.CompetitionStatus {
	switch v {
	case 1:
		return model.CompetitionStatusPending
	case 2:
		return model.CompetitionStatusPublished
	case 3:
		return model.CompetitionStatusRejected
	default:
		return ""
	}
}
