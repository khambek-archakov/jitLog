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

func (r *Repository) GetTraining(ctx context.Context, id int64) (*model.Training, error) {
	const query = `
		select id, user_id, training_date, training_type, duration_minutes, notes, created_at
		from training
		where id = $1
	`

	t, err := scanTraining(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get training by id: %w", err)
	}

	return t, nil
}

// ListTrainings returns up to limit trainings for userID, newest first,
// starting at offset, plus whether more trainings exist past this page.
func (r *Repository) ListTrainings(ctx context.Context, userID int64, limit, offset int) ([]*model.Training, bool, error) {
	const query = `
		select id, user_id, training_date, training_type, duration_minutes, notes, created_at
		from training
		where user_id = $1
		order by training_date desc, id desc
		limit $2 offset $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list trainings: %w", err)
	}
	defer rows.Close()

	var trainings []*model.Training

	for rows.Next() {
		t, err := scanTraining(rows)
		if err != nil {
			return nil, false, fmt.Errorf("scan training: %w", err)
		}

		trainings = append(trainings, t)
	}

	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("list trainings: %w", err)
	}

	hasMore := len(trainings) > limit
	if hasMore {
		trainings = trainings[:limit]
	}

	return trainings, hasMore, nil
}

// ListAllTrainings returns every training for userID, oldest first — for
// stats, which needs the full history (period aggregates and the streak)
// rather than a page of it.
func (r *Repository) ListAllTrainings(ctx context.Context, userID int64) ([]*model.Training, error) {
	const query = `
		select id, user_id, training_date, training_type, duration_minutes, notes, created_at
		from training
		where user_id = $1
		order by training_date asc, id asc
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list all trainings: %w", err)
	}
	defer rows.Close()

	var trainings []*model.Training

	for rows.Next() {
		t, err := scanTraining(rows)
		if err != nil {
			return nil, fmt.Errorf("scan training: %w", err)
		}

		trainings = append(trainings, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list all trainings: %w", err)
	}

	return trainings, nil
}

func (r *Repository) UpdateTraining(
	ctx context.Context,
	id int64,
	date time.Time,
	trainingType model.TrainingType,
	durationMinutes int32,
	notes *string,
) (*model.Training, error) {
	const query = `
		update training
		set training_date = $2, training_type = $3, duration_minutes = $4, notes = $5
		where id = $1
		returning id, user_id, training_date, training_type, duration_minutes, notes, created_at
	`

	t, err := scanTraining(r.db.QueryRow(ctx, query, id, date, trainingTypeToDB(trainingType), durationMinutes, notes))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update training: %w", err)
	}

	return t, nil
}

func (r *Repository) DeleteTraining(ctx context.Context, id int64) error {
	const query = `delete from training where id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete training: %w", err)
	}

	return nil
}

func (r *Repository) GetEditDraftByUserID(ctx context.Context, userID int64) (*model.TrainingEditDraft, error) {
	const query = `
		select user_id, training_id, field, created_at
		from training_edit_draft
		where user_id = $1
	`

	d, err := scanEditDraft(r.db.QueryRow(ctx, query, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get training edit draft by user id: %w", err)
	}

	return d, nil
}

func (r *Repository) SetEditDraft(ctx context.Context, userID, trainingID int64, field model.TrainingEditField) error {
	const query = `
		insert into training_edit_draft (user_id, training_id, field)
		values ($1, $2, $3)
		on conflict (user_id) do update set training_id = $2, field = $3, created_at = now()
	`

	_, err := r.db.Exec(ctx, query, userID, trainingID, editFieldToDB(field))
	if err != nil {
		return fmt.Errorf("set training edit draft: %w", err)
	}

	return nil
}

func (r *Repository) DeleteEditDraft(ctx context.Context, userID int64) error {
	const query = `delete from training_edit_draft where user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete training edit draft: %w", err)
	}

	return nil
}

func scanEditDraft(row pgx.Row) (*model.TrainingEditDraft, error) {
	var (
		d     model.TrainingEditDraft
		field int16
	)

	err := row.Scan(&d.UserID, &d.TrainingID, &field, &d.CreatedAt)
	if err != nil {
		return nil, err
	}

	d.Field = editFieldFromDB(field)

	return &d, nil
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
