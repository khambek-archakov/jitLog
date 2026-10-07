package competition

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

func (r *Repository) GetDraftByUserID(ctx context.Context, userID int64) (*model.UserCompetitionDraft, error) {
	const query = `
		select id, user_id, step, title, date, created_at, updated_at
		from user_competition_draft
		where user_id = $1
	`

	d, err := scanDraft(r.db.QueryRow(ctx, query, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user competition draft by user id: %w", err)
	}

	return d, nil
}

func (r *Repository) CreateDraft(ctx context.Context, userID int64) (*model.UserCompetitionDraft, error) {
	const query = `
		insert into user_competition_draft (user_id)
		values ($1)
		returning id, user_id, step, title, date, created_at, updated_at
	`

	d, err := scanDraft(r.db.QueryRow(ctx, query, userID))
	if err != nil {
		return nil, fmt.Errorf("create user competition draft: %w", err)
	}

	return d, nil
}

func (r *Repository) UpdateDraft(ctx context.Context, d *model.UserCompetitionDraft) error {
	const query = `
		update user_competition_draft
		set step = $2, title = $3, date = $4, updated_at = now()
		where id = $1
	`

	_, err := r.db.Exec(ctx, query, d.ID, draftStepToDB(d.Step), d.Title, d.Date)
	if err != nil {
		return fmt.Errorf("update user competition draft: %w", err)
	}

	return nil
}

func (r *Repository) DeleteDraft(ctx context.Context, userID int64) error {
	const query = `delete from user_competition_draft where user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete user competition draft: %w", err)
	}

	return nil
}

func (r *Repository) CreateUserCompetition(
	ctx context.Context, userID int64, title string, date time.Time,
) (*model.UserCompetition, error) {
	const query = `
		insert into user_competition (user_id, title, date)
		values ($1, $2, $3)
		returning id, user_id, competition_id, title, date, end_date, city, url, result, created_at, updated_at
	`

	c, err := scanUserCompetition(r.db.QueryRow(ctx, query, userID, title, date))
	if err != nil {
		return nil, fmt.Errorf("create user competition: %w", err)
	}

	return c, nil
}

func (r *Repository) GetUserCompetition(ctx context.Context, id int64) (*model.UserCompetition, error) {
	const query = `
		select id, user_id, competition_id, title, date, end_date, city, url, result, created_at, updated_at
		from user_competition
		where id = $1
	`

	c, err := scanUserCompetition(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user competition by id: %w", err)
	}

	return c, nil
}

// ListUpcoming returns every tournament for userID whose COALESCE(end_date,
// date) hasn't passed today yet, soonest first — the main "🏆 Соревнования"
// screen, deliberately unpaginated (unlike ListPast) since an upcoming list
// naturally stays short.
func (r *Repository) ListUpcoming(ctx context.Context, userID int64, today time.Time) ([]*model.UserCompetition, error) {
	const query = `
		select id, user_id, competition_id, title, date, end_date, city, url, result, created_at, updated_at
		from user_competition
		where user_id = $1 and coalesce(end_date, date) >= $2
		order by date asc, id asc
	`

	return r.listUserCompetitions(ctx, query, userID, today)
}

// ListPast returns up to limit tournaments for userID whose COALESCE(end_date,
// date) is before today, newest first, starting at offset, plus whether
// more exist past this page.
func (r *Repository) ListPast(
	ctx context.Context, userID int64, today time.Time, limit, offset int,
) ([]*model.UserCompetition, bool, error) {
	const query = `
		select id, user_id, competition_id, title, date, end_date, city, url, result, created_at, updated_at
		from user_competition
		where user_id = $1 and coalesce(end_date, date) < $2
		order by date desc, id desc
		limit $3 offset $4
	`

	rows, err := r.db.Query(ctx, query, userID, today, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list past user competitions: %w", err)
	}
	defer rows.Close()

	competitions, err := scanUserCompetitions(rows)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(competitions) > limit
	if hasMore {
		competitions = competitions[:limit]
	}

	return competitions, hasMore, nil
}

func (r *Repository) listUserCompetitions(
	ctx context.Context, query string, args ...any,
) ([]*model.UserCompetition, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list user competitions: %w", err)
	}
	defer rows.Close()

	return scanUserCompetitions(rows)
}

func (r *Repository) UpdateUserCompetition(
	ctx context.Context,
	id int64,
	title string,
	date time.Time,
	endDate *time.Time,
	city, url, result *string,
) (*model.UserCompetition, error) {
	const query = `
		update user_competition
		set title = $2, date = $3, end_date = $4, city = $5, url = $6, result = $7, updated_at = now()
		where id = $1
		returning id, user_id, competition_id, title, date, end_date, city, url, result, created_at, updated_at
	`

	c, err := scanUserCompetition(r.db.QueryRow(ctx, query, id, title, date, endDate, city, url, result))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update user competition: %w", err)
	}

	return c, nil
}

func (r *Repository) DeleteUserCompetition(ctx context.Context, id int64) error {
	const query = `delete from user_competition where id = $1`

	_, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete user competition: %w", err)
	}

	return nil
}

func (r *Repository) GetEditDraftByUserID(ctx context.Context, userID int64) (*model.UserCompetitionEditDraft, error) {
	const query = `
		select user_id, user_competition_id, field, created_at
		from user_competition_edit_draft
		where user_id = $1
	`

	d, err := scanEditDraft(r.db.QueryRow(ctx, query, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get user competition edit draft by user id: %w", err)
	}

	return d, nil
}

func (r *Repository) SetEditDraft(
	ctx context.Context, userID, userCompetitionID int64, field model.UserCompetitionEditField,
) error {
	const query = `
		insert into user_competition_edit_draft (user_id, user_competition_id, field)
		values ($1, $2, $3)
		on conflict (user_id) do update set user_competition_id = $2, field = $3, created_at = now()
	`

	_, err := r.db.Exec(ctx, query, userID, userCompetitionID, editFieldToDB(field))
	if err != nil {
		return fmt.Errorf("set user competition edit draft: %w", err)
	}

	return nil
}

func (r *Repository) DeleteEditDraft(ctx context.Context, userID int64) error {
	const query = `delete from user_competition_edit_draft where user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete user competition edit draft: %w", err)
	}

	return nil
}

func scanDraft(row pgx.Row) (*model.UserCompetitionDraft, error) {
	var (
		d    model.UserCompetitionDraft
		step int16
	)

	err := row.Scan(&d.ID, &d.UserID, &step, &d.Title, &d.Date, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}

	d.Step = draftStepFromDB(step)

	return &d, nil
}

func scanEditDraft(row pgx.Row) (*model.UserCompetitionEditDraft, error) {
	var (
		d     model.UserCompetitionEditDraft
		field int16
	)

	err := row.Scan(&d.UserID, &d.UserCompetitionID, &field, &d.CreatedAt)
	if err != nil {
		return nil, err
	}

	d.Field = editFieldFromDB(field)

	return &d, nil
}

func scanUserCompetition(row pgx.Row) (*model.UserCompetition, error) {
	var c model.UserCompetition

	err := row.Scan(
		&c.ID, &c.UserID, &c.CompetitionID, &c.Title, &c.Date, &c.EndDate, &c.City, &c.URL, &c.Result,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &c, nil
}

func scanUserCompetitions(rows pgx.Rows) ([]*model.UserCompetition, error) {
	var competitions []*model.UserCompetition

	for rows.Next() {
		c, err := scanUserCompetition(rows)
		if err != nil {
			return nil, fmt.Errorf("scan user competition: %w", err)
		}

		competitions = append(competitions, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list user competitions: %w", err)
	}

	return competitions, nil
}
