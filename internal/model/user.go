package model

import "time"

type User struct {
	ID             int64
	TelegramID     int64
	Name           *string
	Age            *int16
	Belt           Belt
	OnboardingStep OnboardingStep
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
