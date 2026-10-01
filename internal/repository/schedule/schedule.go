package schedule

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

func (r *Repository) GetDraftByUserID(ctx context.Context, userID int64) (*model.ScheduleDraft, error) {
	const query = `
		select id, user_id, step, day_of_week, time_minutes, training_type, created_at, updated_at
		from schedule_draft
		where user_id = $1
	`

	d, err := scanDraft(r.db.QueryRow(ctx, query, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get schedule draft by user id: %w", err)
	}

	return d, nil
}

func (r *Repository) CreateDraft(ctx context.Context, userID int64) (*model.ScheduleDraft, error) {
	const query = `
		insert into schedule_draft (user_id)
		values ($1)
		returning id, user_id, step, day_of_week, time_minutes, training_type, created_at, updated_at
	`

	d, err := scanDraft(r.db.QueryRow(ctx, query, userID))
	if err != nil {
		return nil, fmt.Errorf("create schedule draft: %w", err)
	}

	return d, nil
}

func (r *Repository) UpdateDraft(ctx context.Context, d *model.ScheduleDraft) error {
	const query = `
		update schedule_draft
		set step = $2, day_of_week = $3, time_minutes = $4, training_type = $5, updated_at = now()
		where id = $1
	`

	_, err := r.db.Exec(ctx, query,
		d.ID, draftStepToDB(d.Step), d.DayOfWeek, d.TimeMinutes, trainingTypeToDB(d.TrainingType),
	)
	if err != nil {
		return fmt.Errorf("update schedule draft: %w", err)
	}

	return nil
}

func (r *Repository) DeleteDraft(ctx context.Context, userID int64) error {
	const query = `delete from schedule_draft where user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete schedule draft: %w", err)
	}

	return nil
}

func (r *Repository) CreateSlot(
	ctx context.Context, userID int64, dayOfWeek, timeMinutes int16, trainingType model.TrainingType,
) (*model.ScheduleSlot, error) {
	const query = `
		insert into schedule_slot (user_id, day_of_week, time_minutes, training_type)
		values ($1, $2, $3, $4)
		returning id, user_id, day_of_week, time_minutes, training_type, created_at
	`

	s, err := scanSlot(r.db.QueryRow(ctx, query, userID, dayOfWeek, timeMinutes, trainingTypeToDB(trainingType)))
	if err != nil {
		return nil, fmt.Errorf("create schedule slot: %w", err)
	}

	return s, nil
}

// ListSlots returns every slot for userID, ordered the way a week reads:
// Monday first, earliest time first within a day.
func (r *Repository) ListSlots(ctx context.Context, userID int64) ([]*model.ScheduleSlot, error) {
	const query = `
		select id, user_id, day_of_week, time_minutes, training_type, created_at
		from schedule_slot
		where user_id = $1
		order by day_of_week asc, time_minutes asc
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list schedule slots: %w", err)
	}
	defer rows.Close()

	var slots []*model.ScheduleSlot

	for rows.Next() {
		s, err := scanSlot(rows)
		if err != nil {
			return nil, fmt.Errorf("scan schedule slot: %w", err)
		}

		slots = append(slots, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list schedule slots: %w", err)
	}

	return slots, nil
}

func (r *Repository) GetSlot(ctx context.Context, id int64) (*model.ScheduleSlot, error) {
	const query = `
		select id, user_id, day_of_week, time_minutes, training_type, created_at
		from schedule_slot
		where id = $1
	`

	s, err := scanSlot(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get schedule slot by id: %w", err)
	}

	return s, nil
}

func (r *Repository) UpdateSlot(
	ctx context.Context, id int64, dayOfWeek, timeMinutes int16, trainingType model.TrainingType,
) (*model.ScheduleSlot, error) {
	const query = `
		update schedule_slot
		set day_of_week = $2, time_minutes = $3, training_type = $4
		where id = $1
		returning id, user_id, day_of_week, time_minutes, training_type, created_at
	`

	s, err := scanSlot(r.db.QueryRow(ctx, query, id, dayOfWeek, timeMinutes, trainingTypeToDB(trainingType)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update schedule slot: %w", err)
	}

	return s, nil
}

func (r *Repository) DeleteSlot(ctx context.Context, id int64) error {
	const query = `delete from schedule_slot where id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete schedule slot: %w", err)
	}

	return nil
}

func (r *Repository) GetEditDraftByUserID(ctx context.Context, userID int64) (*model.ScheduleEditDraft, error) {
	const query = `
		select user_id, slot_id, created_at
		from schedule_edit_draft
		where user_id = $1
	`

	d, err := scanEditDraft(r.db.QueryRow(ctx, query, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get schedule edit draft by user id: %w", err)
	}

	return d, nil
}

func (r *Repository) SetEditDraft(ctx context.Context, userID, slotID int64) error {
	const query = `
		insert into schedule_edit_draft (user_id, slot_id)
		values ($1, $2)
		on conflict (user_id) do update set slot_id = $2, created_at = now()
	`

	_, err := r.db.Exec(ctx, query, userID, slotID)
	if err != nil {
		return fmt.Errorf("set schedule edit draft: %w", err)
	}

	return nil
}

func (r *Repository) DeleteEditDraft(ctx context.Context, userID int64) error {
	const query = `delete from schedule_edit_draft where user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete schedule edit draft: %w", err)
	}

	return nil
}

func scanEditDraft(row pgx.Row) (*model.ScheduleEditDraft, error) {
	var d model.ScheduleEditDraft

	err := row.Scan(&d.UserID, &d.SlotID, &d.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func scanDraft(row pgx.Row) (*model.ScheduleDraft, error) {
	var (
		d            model.ScheduleDraft
		step         int16
		trainingType int16
	)

	err := row.Scan(&d.ID, &d.UserID, &step, &d.DayOfWeek, &d.TimeMinutes, &trainingType, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	d.Step = draftStepFromDB(step)
	d.TrainingType = trainingTypeFromDB(trainingType)

	return &d, nil
}

func scanSlot(row pgx.Row) (*model.ScheduleSlot, error) {
	var (
		s            model.ScheduleSlot
		trainingType int16
	)

	err := row.Scan(&s.ID, &s.UserID, &s.DayOfWeek, &s.TimeMinutes, &trainingType, &s.CreatedAt)
	if err != nil {
		return nil, err
	}

	s.TrainingType = trainingTypeFromDB(trainingType)

	return &s, nil
}
