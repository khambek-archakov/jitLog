package training

import "github.com/khambek-archakov/jitLog/internal/model"

func trainingTypeToDB(t model.TrainingType) int16 {
	switch t {
	case model.TrainingTypeGi:
		return 1
	case model.TrainingTypeNoGi:
		return 2
	case model.TrainingTypeOpenMat:
		return 3
	default:
		return 0
	}
}

func trainingTypeFromDB(v int16) model.TrainingType {
	switch v {
	case 1:
		return model.TrainingTypeGi
	case 2:
		return model.TrainingTypeNoGi
	case 3:
		return model.TrainingTypeOpenMat
	default:
		return model.TrainingTypeNone
	}
}

// draftStepToDB leaves ordinal 3 retired (it used to mean the now-removed
// awaiting_notes step) rather than reusing it — a stale draft row still
// sitting at 3 when this ships falls through draftStepFromDB's default
// case back to awaiting_date instead of aliasing onto some other step.
func draftStepToDB(s model.TrainingDraftStep) int16 {
	switch s {
	case model.TrainingDraftStepAwaitingType:
		return 1
	case model.TrainingDraftStepAwaitingDuration:
		return 2
	default:
		return 0
	}
}

func draftStepFromDB(v int16) model.TrainingDraftStep {
	switch v {
	case 1:
		return model.TrainingDraftStepAwaitingType
	case 2:
		return model.TrainingDraftStepAwaitingDuration
	default:
		return model.TrainingDraftStepAwaitingDate
	}
}

func editFieldToDB(f model.TrainingEditField) int16 {
	switch f {
	case model.TrainingEditFieldDuration:
		return 1
	case model.TrainingEditFieldNotes:
		return 2
	case model.TrainingEditFieldRounds:
		return 3
	default:
		return 0
	}
}

func editFieldFromDB(v int16) model.TrainingEditField {
	switch v {
	case 1:
		return model.TrainingEditFieldDuration
	case 2:
		return model.TrainingEditFieldNotes
	case 3:
		return model.TrainingEditFieldRounds
	default:
		return ""
	}
}
