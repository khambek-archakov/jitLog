package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/repository/tx"
)

// pgUniqueViolation is Postgres's standard SQLSTATE code for a unique
// constraint violation (23505) — stable across versions, no extra
// dependency needed just to name it.
const pgUniqueViolation = "23505"

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	const query = `
		select id, telegram_id, name, age, belt, onboarding_step, timezone, city, reminders_enabled,
			last_activity_at, winback_sent_at, winback_declined_at, created_at, updated_at
		from "user"
		where telegram_id = $1
	`

	u, err := scanUser(r.db.QueryRow(ctx, query, telegramID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by telegram id: %w", err)
	}

	return u, nil
}

// GetByID resolves a user_id to their full row — used wherever a
// notification needs a TelegramID but only a user_id is on hand (e.g.
// catalog/moderate notifying every owner of a user_competition row that
// got relinked, not just the original submitter).
func (r *Repository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	const query = `
		select id, telegram_id, name, age, belt, onboarding_step, timezone, city, reminders_enabled,
			last_activity_at, winback_sent_at, winback_declined_at, created_at, updated_at
		from "user"
		where id = $1
	`

	u, err := scanUser(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}

	return u, nil
}

func (r *Repository) Create(ctx context.Context, telegramID int64) (*model.User, error) {
	const query = `
		insert into "user" (telegram_id)
		values ($1)
		returning id, telegram_id, name, age, belt, onboarding_step, timezone, city, reminders_enabled,
			last_activity_at, winback_sent_at, winback_declined_at, created_at, updated_at
	`

	u, err := scanUser(r.db.QueryRow(ctx, query, telegramID))
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return u, nil
}

func (r *Repository) Update(ctx context.Context, u *model.User) error {
	const query = `
		update "user"
		set name = $2, age = $3, belt = $4, onboarding_step = $5, timezone = $6, city = $7,
			reminders_enabled = $8, updated_at = now()
		where id = $1
	`

	_, err := r.db.Exec(
		ctx, query, u.ID, u.Name, u.Age, beltToDB(u.Belt), onboardingStepToDB(u.OnboardingStep), u.Timezone, u.City,
		u.RemindersEnabled,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
}

// ListRemindersEnabled returns every user with reminders_enabled set —
// internal/usecase/reminder's own ticker walks this list each tick rather
// than scanning the whole "user" table.
func (r *Repository) ListRemindersEnabled(ctx context.Context) ([]*model.User, error) {
	const query = `
		select id, telegram_id, name, age, belt, onboarding_step, timezone, city, reminders_enabled,
			last_activity_at, winback_sent_at, winback_declined_at, created_at, updated_at
		from "user"
		where reminders_enabled = true
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list reminders enabled users: %w", err)
	}
	defer rows.Close()

	var users []*model.User

	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}

		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list reminders enabled users: %w", err)
	}

	return users, nil
}

// ListAll returns every user — internal/usecase/winback's own ticker walks
// this list each tick, since (unlike reminders) almost any user is a
// winback candidate and the actual gating (silence/resend/local-hours
// windows) is date math better kept in Go, where it's unit-testable with a
// fake clock, than folded into a WHERE clause.
func (r *Repository) ListAll(ctx context.Context) ([]*model.User, error) {
	const query = `
		select id, telegram_id, name, age, belt, onboarding_step, timezone, city, reminders_enabled,
			last_activity_at, winback_sent_at, winback_declined_at, created_at, updated_at
		from "user"
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list all users: %w", err)
	}
	defer rows.Close()

	var users []*model.User

	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}

		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list all users: %w", err)
	}

	return users, nil
}

// TouchActivity records at as userID's last_activity_at — see
// internal/usecase/router's own activity middleware, the only caller.
// Deliberately its own narrow statement rather than routed through the
// general-purpose Update, which writes a whole stale in-memory *model.User
// back and would otherwise race/clobber whatever this (or SetWinbackSent/
// SetWinbackDeclined) just wrote concurrently.
func (r *Repository) TouchActivity(ctx context.Context, userID int64, at time.Time) error {
	const query = `update "user" set last_activity_at = $2 where id = $1`

	_, err := r.db.Exec(ctx, query, userID, at)
	if err != nil {
		return fmt.Errorf("touch user activity: %w", err)
	}

	return nil
}

// SetWinbackSent records at as userID's winback_sent_at — see
// internal/usecase/winback. Same narrow-statement reasoning as
// TouchActivity.
func (r *Repository) SetWinbackSent(ctx context.Context, userID int64, at time.Time) error {
	const query = `update "user" set winback_sent_at = $2 where id = $1`

	_, err := r.db.Exec(ctx, query, userID, at)
	if err != nil {
		return fmt.Errorf("set winback sent: %w", err)
	}

	return nil
}

// SetWinbackDeclined records at as userID's winback_declined_at — see
// internal/usecase/winback. Same narrow-statement reasoning as
// TouchActivity.
func (r *Repository) SetWinbackDeclined(ctx context.Context, userID int64, at time.Time) error {
	const query = `update "user" set winback_declined_at = $2 where id = $1`

	_, err := r.db.Exec(ctx, query, userID, at)
	if err != nil {
		return fmt.Errorf("set winback declined: %w", err)
	}

	return nil
}

// ChangeBelt updates userID's belt and, when recordPromotion is true,
// records a fresh belt_promotion row for newBelt dated promotedAt — both
// in one transaction, so a crash between the two can never leave the
// user's belt pointing at a belt with no promotion row backing it (or
// vice versa). recorded reports whether that row was actually newly
// created: always false when recordPromotion is false (the caller's own
// call — e.g. a downgrade or re-picking the current belt — decided this
// isn't a promotion worth logging), and also false when the row already
// existed (a repeat promotion to a belt once held before).
// internal/usecase/profile's own belt-change flow is the only caller —
// onboarding's first-ever belt still goes through the plain
// Update+AddBeltPromotion pair, since there's nothing to protect there.
func (r *Repository) ChangeBelt(
	ctx context.Context, userID int64, newBelt model.Belt, promotedAt time.Time, recordPromotion bool,
) (bool, error) {
	var recorded bool

	err := tx.WithTx(ctx, r.db, func(ctx context.Context) error {
		q := tx.QuerierFromContext(ctx, r.db)

		if _, err := q.Exec(ctx, `update "user" set belt = $2, updated_at = now() where id = $1`, userID, beltToDB(newBelt)); err != nil {
			return fmt.Errorf("update user belt: %w", err)
		}

		if !recordPromotion {
			return nil
		}

		_, err := q.Exec(
			ctx, `insert into belt_promotion (user_id, belt, promoted_at) values ($1, $2, $3)`,
			userID, beltToDB(newBelt), promotedAt,
		)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
				return nil
			}

			return fmt.Errorf("add belt promotion: %w", err)
		}

		recorded = true

		return nil
	})
	if err != nil {
		return false, err
	}

	return recorded, nil
}

// AddBeltPromotion records a belt as having started on promotedAt. Each
// belt can only be recorded once per user (see the unique constraint on
// belt_promotion) — ErrDuplicateBeltPromotion signals that this belt was
// already recorded for this user, which callers should turn into a toast
// rather than a hard failure.
func (r *Repository) AddBeltPromotion(ctx context.Context, userID int64, belt model.Belt, promotedAt time.Time) error {
	const query = `
		insert into belt_promotion (user_id, belt, promoted_at)
		values ($1, $2, $3)
	`

	_, err := r.db.Exec(ctx, query, userID, beltToDB(belt), promotedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return model.ErrDuplicateBeltPromotion
		}

		return fmt.Errorf("add belt promotion: %w", err)
	}

	return nil
}

// ListBeltPromotions returns every belt promotion for userID, oldest
// first — the order an "as of" lookup needs.
func (r *Repository) ListBeltPromotions(ctx context.Context, userID int64) ([]*model.BeltPromotion, error) {
	const query = `
		select id, user_id, belt, promoted_at, created_at
		from belt_promotion
		where user_id = $1
		order by promoted_at asc
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list belt promotions: %w", err)
	}
	defer rows.Close()

	var promotions []*model.BeltPromotion

	for rows.Next() {
		p, err := scanBeltPromotion(rows)
		if err != nil {
			return nil, fmt.Errorf("scan belt promotion: %w", err)
		}

		promotions = append(promotions, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list belt promotions: %w", err)
	}

	return promotions, nil
}

// GetEditDraftByUserID looks up the pending profile edit draft for userID,
// if any — its mere existence means "the next free-text message from this
// user is their new age."
func (r *Repository) GetEditDraftByUserID(ctx context.Context, userID int64) (*model.ProfileEditDraft, error) {
	const query = `
		select user_id, created_at
		from profile_edit_draft
		where user_id = $1
	`

	d, err := scanEditDraft(r.db.QueryRow(ctx, query, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get profile edit draft by user id: %w", err)
	}

	return d, nil
}

func (r *Repository) SetEditDraft(ctx context.Context, userID int64) error {
	const query = `
		insert into profile_edit_draft (user_id)
		values ($1)
		on conflict (user_id) do update set created_at = now()
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("set profile edit draft: %w", err)
	}

	return nil
}

func (r *Repository) DeleteEditDraft(ctx context.Context, userID int64) error {
	const query = `delete from profile_edit_draft where user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete profile edit draft: %w", err)
	}

	return nil
}

func scanEditDraft(row pgx.Row) (*model.ProfileEditDraft, error) {
	var d model.ProfileEditDraft

	err := row.Scan(&d.UserID, &d.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func scanBeltPromotion(row pgx.Row) (*model.BeltPromotion, error) {
	var (
		p    model.BeltPromotion
		belt int16
	)

	err := row.Scan(&p.ID, &p.UserID, &belt, &p.PromotedAt, &p.CreatedAt)
	if err != nil {
		return nil, err
	}

	p.Belt = beltFromDB(belt)

	return &p, nil
}

func scanUser(row pgx.Row) (*model.User, error) {
	var (
		u    model.User
		belt int16
		step int16
	)

	err := row.Scan(
		&u.ID,
		&u.TelegramID,
		&u.Name,
		&u.Age,
		&belt,
		&step,
		&u.Timezone,
		&u.City,
		&u.RemindersEnabled,
		&u.LastActivityAt,
		&u.WinbackSentAt,
		&u.WinbackDeclinedAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	u.Belt = beltFromDB(belt)
	u.OnboardingStep = onboardingStepFromDB(step)

	return &u, nil
}
