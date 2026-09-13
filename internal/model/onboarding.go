package model

type OnboardingStep string

const (
	OnboardingStepAwaitingName OnboardingStep = "awaiting_name"
	OnboardingStepAwaitingAge  OnboardingStep = "awaiting_age"
	OnboardingStepAwaitingBelt OnboardingStep = "awaiting_belt"
	OnboardingStepCompleted    OnboardingStep = "completed"
)
