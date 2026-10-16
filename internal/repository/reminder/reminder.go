// Package reminder owns the reminder_sent table — the once-per-week-per-user
// idempotency marker internal/usecase/reminder relies on instead of any
// in-memory state, so a process restart mid-window still can't send twice.
package reminder

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
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

// HasSent reports whether userID already has a reminder_sent row for
// weekStart.
func (r *Repository) HasSent(ctx context.Context, userID int64, weekStart time.Time) (bool, error) {
	const query = `select exists(select 1 from reminder_sent where user_id = $1 and week_start = $2)`

	var exists bool

	if err := r.db.QueryRow(ctx, query, userID, weekStart).Scan(&exists); err != nil {
		return false, fmt.Errorf("check reminder sent: %w", err)
	}

	return exists, nil
}

// HasSentWithin reports whether userID has any reminder_sent row at or
// after since — internal/usecase/winback's own "don't pile a winback nudge
// on top of a just-sent weekly reminder" check.
func (r *Repository) HasSentWithin(ctx context.Context, userID int64, since time.Time) (bool, error) {
	const query = `select exists(select 1 from reminder_sent where user_id = $1 and sent_at >= $2)`

	var exists bool

	if err := r.db.QueryRow(ctx, query, userID, since).Scan(&exists); err != nil {
		return false, fmt.Errorf("check reminder sent within: %w", err)
	}

	return exists, nil
}

// MarkSent records that userID's reminder for weekStart has been sent. A
// duplicate insert (two overlapping runs racing the same user/week) is not
// an error — whichever one lands first wins, the other is a no-op.
func (r *Repository) MarkSent(ctx context.Context, userID int64, weekStart time.Time) error {
	const query = `
		insert into reminder_sent (user_id, week_start)
		values ($1, $2)
		on conflict (user_id, week_start) do nothing
	`

	_, err := r.db.Exec(ctx, query, userID, weekStart)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil
		}

		return fmt.Errorf("mark reminder sent: %w", err)
	}

	return nil
}
