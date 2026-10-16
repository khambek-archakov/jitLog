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
	Timezone *string
	// City is the catalog's default filter (see
	// internal/usecase/catalog/list) — nullable, and like Timezone nothing
	// sets it yet. A per-view filter change in the catalog screen never
	// writes back here.
	City *string
	// RemindersEnabled gates the weekly training reminder (see
	// internal/usecase/reminder) — it only ever fires once Timezone is also
	// set and at least one schedule slot exists.
	RemindersEnabled bool
	// LastActivityAt is updated on every incoming update (message,
	// callback, command) — see internal/usecase/router's own activity
	// middleware — throttled to at most once a minute, except it's always
	// written the moment it would lift an active WinbackDeclinedAt.
	// Existing rows got the migration's own timestamp, not the epoch, so
	// nobody looks silent the instant this column was added.
	LastActivityAt time.Time
	// WinbackSentAt/WinbackDeclinedAt back the "haven't seen you in a
	// while" nudge (see internal/usecase/winback) — both nullable, nothing
	// sets them until a user goes quiet for 14+ days.
	WinbackSentAt     *time.Time
	WinbackDeclinedAt *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
