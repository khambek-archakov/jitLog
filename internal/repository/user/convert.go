package user

import "github.com/khambek-archakov/jitLog/internal/model"

func beltToDB(b model.Belt) int16 {
	switch b {
	case model.BeltWhite:
		return 1
	case model.BeltBlue:
		return 2
	case model.BeltPurple:
		return 3
	case model.BeltBrown:
		return 4
	case model.BeltBlack:
		return 5
	default:
		return 0
	}
}

func beltFromDB(v int16) model.Belt {
	switch v {
	case 1:
		return model.BeltWhite
	case 2:
		return model.BeltBlue
	case 3:
		return model.BeltPurple
	case 4:
		return model.BeltBrown
	case 5:
		return model.BeltBlack
	default:
		return model.BeltNone
	}
}

func onboardingStepToDB(s model.OnboardingStep) int16 {
	switch s {
	case model.OnboardingStepAwaitingAge:
		return 1
	case model.OnboardingStepAwaitingBelt:
		return 2
	case model.OnboardingStepCompleted:
		return 3
	default:
		return 0
	}
}

func onboardingStepFromDB(v int16) model.OnboardingStep {
	switch v {
	case 1:
		return model.OnboardingStepAwaitingAge
	case 2:
		return model.OnboardingStepAwaitingBelt
	case 3:
		return model.OnboardingStepCompleted
	default:
		return model.OnboardingStepAwaitingName
	}
}
