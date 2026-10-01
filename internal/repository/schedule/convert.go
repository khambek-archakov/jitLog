package schedule

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

func draftStepToDB(s model.ScheduleDraftStep) int16 {
	switch s {
	case model.ScheduleDraftStepAwaitingTime:
		return 1
	case model.ScheduleDraftStepAwaitingType:
		return 2
	default:
		return 0
	}
}

func draftStepFromDB(v int16) model.ScheduleDraftStep {
	switch v {
	case 1:
		return model.ScheduleDraftStepAwaitingTime
	case 2:
		return model.ScheduleDraftStepAwaitingType
	default:
		return model.ScheduleDraftStepAwaitingDay
	}
}
