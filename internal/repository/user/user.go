package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khambek-archakov/jitLog/internal/model"
)

type Repository struct {
	db *pgxpool.Pool
}

func New(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetByTelegramID(ctx context.Context, telegramID int64) (*model.User, error) {
	const query = `
		select id, telegram_id, name, age, belt, onboarding_step, created_at, updated_at
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
		returning id, telegram_id, name, age, belt, onboarding_step, created_at, updated_at
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
		set name = $2, age = $3, belt = $4, onboarding_step = $5, updated_at = now()
		where id = $1
	`

	_, err := r.db.Exec(ctx, query, u.ID, u.Name, u.Age, beltToDB(u.Belt), onboardingStepToDB(u.OnboardingStep))
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return nil
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
