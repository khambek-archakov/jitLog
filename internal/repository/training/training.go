package training

import (
	"context"
	"errors"
	"fmt"
	"time"

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

func (r *Repository) GetDraftByUserID(ctx context.Context, userID int64) (*model.TrainingDraft, error) {
	const query = `
		select id, user_id, step, training_date, training_type, duration_minutes, notes, created_at, updated_at
		from training_draft
		where user_id = $1
	`

	d, err := scanDraft(r.db.QueryRow(ctx, query, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get training draft by user id: %w", err)
	}

	return d, nil
}

func (r *Repository) CreateDraft(ctx context.Context, userID int64) (*model.TrainingDraft, error) {
	const query = `
		insert into training_draft (user_id)
		values ($1)
		returning id, user_id, step, training_date, training_type, duration_minutes, notes, created_at, updated_at
	`

	d, err := scanDraft(r.db.QueryRow(ctx, query, userID))
	if err != nil {
		return nil, fmt.Errorf("create training draft: %w", err)
	}

	return d, nil
}

func (r *Repository) UpdateDraft(ctx context.Context, d *model.TrainingDraft) error {
	const query = `
		update training_draft
		set step = $2, training_date = $3, training_type = $4, duration_minutes = $5, notes = $6, updated_at = now()
		where id = $1
	`

	_, err := r.db.Exec(ctx, query,
		d.ID, draftStepToDB(d.Step), d.Date, trainingTypeToDB(d.TrainingType), d.DurationMinutes, d.Notes,
	)
	if err != nil {
		return fmt.Errorf("update training draft: %w", err)
	}

	return nil
}

func (r *Repository) DeleteDraft(ctx context.Context, userID int64) error {
	const query = `delete from training_draft where user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete training draft: %w", err)
	}

	return nil
}

func (r *Repository) CreateTraining(
	ctx context.Context,
	userID int64,
	date time.Time,
	trainingType model.TrainingType,
	durationMinutes int32,
	notes *string,
) (*model.Training, error) {
	const query = `
		insert into training (user_id, training_date, training_type, duration_minutes, notes)
		values ($1, $2, $3, $4, $5)
		returning id, user_id, training_date, training_type, duration_minutes, notes, created_at
	`

	t, err := scanTraining(r.db.QueryRow(ctx, query, userID, date, trainingTypeToDB(trainingType), durationMinutes, notes))
	if err != nil {
		return nil, fmt.Errorf("create training: %w", err)
	}

	return t, nil
}

func scanDraft(row pgx.Row) (*model.TrainingDraft, error) {
	var (
		d            model.TrainingDraft
		step         int16
		trainingType int16
	)

	err := row.Scan(
		&d.ID,
		&d.UserID,
		&step,
		&d.Date,
		&trainingType,
		&d.DurationMinutes,
		&d.Notes,
		&d.CreatedAt,
		&d.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	d.Step = draftStepFromDB(step)
	d.TrainingType = trainingTypeFromDB(trainingType)

	return &d, nil
}

func scanTraining(row pgx.Row) (*model.Training, error) {
	var (
		t            model.Training
		trainingType int16
	)

	err := row.Scan(
		&t.ID,
		&t.UserID,
		&t.Date,
		&trainingType,
		&t.DurationMinutes,
		&t.Notes,
		&t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	t.TrainingType = trainingTypeFromDB(trainingType)

	return &t, nil
}
