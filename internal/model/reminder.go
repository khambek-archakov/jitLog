package model

// ReminderKind distinguishes the weekly training reminder from the
// post-session "record it?" nudge — both share one reminder_log table
// (see internal/repository/reminder) for their idempotency/rate-limit
// bookkeeping.
type ReminderKind string

const (
	ReminderKindWeekly      ReminderKind = "weekly"
	ReminderKindPostSession ReminderKind = "post_session"
)
