package model

import "time"

type User struct {
	ID             int64
	TelegramID     int64
	Name           *string
	Age            *int16
	Belt           Belt
	OnboardingStep OnboardingStep
	// Timezone is an IANA name (e.g. "Europe/Moscow"), nullable — nothing
	// sets it yet (no onboarding step or profile field for it), so every
	// reader must treat nil (or a value time.LoadLocation rejects) as UTC.
	Timezone  *string
	CreatedAt time.Time
	UpdatedAt time.Time
}
