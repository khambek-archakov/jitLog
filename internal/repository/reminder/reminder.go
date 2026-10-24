// Package reminder owns the reminder_log table — the idempotency/rate-limit
// marker both internal/usecase/reminder's weekly nudge and its
// post-session nudge write to, instead of any in-memory state, so a
// process restart mid-window still can't send twice.
package reminder

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khambek-archakov/jitLog/internal/model"
)

// pgUniqueViolation mirrors internal/repository/user's own private
// constant — Postgres's standard SQLSTATE code for a unique violation.
const pgUniqueViolation = "23505"

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func reminderKindToDB(k model.ReminderKind) int16 {
	switch k {
	case model.ReminderKindWeekly:
		return 1
	case model.ReminderKindPostSession:
		return 2
	default:
		return 0
	}
}

// HasSent reports whether a reminder_log row already exists for this
// exact (user, kind, ref_date, slot_id) — weekly's own idempotency check
// (slotID always 0) and post_session's own (one per slot per day).
func (r *Repository) HasSent(
	ctx context.Context, userID int64, kind model.ReminderKind, refDate time.Time, slotID int64,
) (bool, error) {
	const query = `
		select exists(select 1 from reminder_log where user_id = $1 and kind = $2 and ref_date = $3 and slot_id = $4)
	`

	var exists bool

	if err := r.db.QueryRow(ctx, query, userID, reminderKindToDB(kind), refDate, slotID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check reminder sent: %w", err)
	}

	return exists, nil
}

// HasSentWithin reports whether userID has any reminder_log row (weekly or
// post_session) at or after since — internal/usecase/winback's own "don't
// pile a winback nudge on top of a just-sent reminder" check.
func (r *Repository) HasSentWithin(ctx context.Context, userID int64, since time.Time) (bool, error) {
	const query = `select exists(select 1 from reminder_log where user_id = $1 and sent_at >= $2)`

	var exists bool

	if err := r.db.QueryRow(ctx, query, userID, since).Scan(&exists); err != nil {
		return false, fmt.Errorf("check reminder sent within: %w", err)
	}

	return exists, nil
}

// CountSentBetween counts every reminder_log row (any kind) sent for
// userID in [from, to) — the post-session job's own "at most two
// reminders per local calendar day, weekly and post-session combined" cap.
func (r *Repository) CountSentBetween(ctx context.Context, userID int64, from, to time.Time) (int, error) {
	const query = `select count(*) from reminder_log where user_id = $1 and sent_at >= $2 and sent_at < $3`

	var n int

	if err := r.db.QueryRow(ctx, query, userID, from, to).Scan(&n); err != nil {
		return 0, fmt.Errorf("count reminders sent: %w", err)
	}

	return n, nil
}

// LastMessageID returns the most recently sent reminder's message_id for
// userID (any kind), if any — the post-session job's own "strip the
// previous reminder's keyboard before sending a new one" step.
func (r *Repository) LastMessageID(ctx context.Context, userID int64) (int64, bool, error) {
	const query = `
		select message_id from reminder_log
		where user_id = $1 and message_id is not null
		order by sent_at desc
		limit 1
	`

	var id int64

	err := r.db.QueryRow(ctx, query, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("get last reminder message id: %w", err)
	}

	return id, true, nil
}

// MarkSent records that a reminder was sent — including the message id,
// so a later reminder (weekly or post-session) can strip its keyboard. A
// duplicate insert (two overlapping runs racing the same user/kind/date/
// slot) is not an error — whichever one lands first wins, the other is a
// no-op.
func (r *Repository) MarkSent(
	ctx context.Context, userID int64, kind model.ReminderKind, refDate time.Time, slotID, messageID int64,
) error {
	const query = `
		insert into reminder_log (user_id, kind, ref_date, slot_id, message_id)
		values ($1, $2, $3, $4, $5)
		on conflict (user_id, kind, ref_date, slot_id) do nothing
	`

	_, err := r.db.Exec(ctx, query, userID, reminderKindToDB(kind), refDate, slotID, messageID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil
		}

		return fmt.Errorf("mark reminder sent: %w", err)
	}

	return nil
}
