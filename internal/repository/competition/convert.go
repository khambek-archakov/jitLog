package competition

import "github.com/khambek-archakov/jitLog/internal/model"

func draftStepToDB(s model.UserCompetitionDraftStep) int16 {
	switch s {
	case model.UserCompetitionDraftStepAwaitingDate:
		return 1
	default:
		return 0
	}
}

func draftStepFromDB(v int16) model.UserCompetitionDraftStep {
	switch v {
	case 1:
		return model.UserCompetitionDraftStepAwaitingDate
	default:
		return model.UserCompetitionDraftStepAwaitingTitle
	}
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
