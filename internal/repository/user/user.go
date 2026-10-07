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
		select id, telegram_id, name, age, belt, onboarding_step, timezone, created_at, updated_at
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

func (r *Repository) Create(ctx context.Context, telegramID int64) (*model.User, error) {
	const query = `
		insert into "user" (telegram_id)
		values ($1)
		returning id, telegram_id, name, age, belt, onboarding_step, timezone, created_at, updated_at
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
		set name = $2, age = $3, belt = $4, onboarding_step = $5, timezone = $6, updated_at = now()
		where id = $1
	`

	_, err := r.db.Exec(
		ctx, query, u.ID, u.Name, u.Age, beltToDB(u.Belt), onboardingStepToDB(u.OnboardingStep), u.Timezone,
	)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
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
