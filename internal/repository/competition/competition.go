package competition

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/repository/tx"
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

// CreateUserCompetitionFromCatalog is "➕ В мои" — copies a catalog entry's
// fields into a brand new personal record and links it via competitionID,
// so it behaves exactly like any other personal record from then on
// (editable, deletable, submittable again, independent of the catalog
// entry it was copied from).
func (r *Repository) CreateUserCompetitionFromCatalog(
	ctx context.Context, userID, competitionID int64, title string, date time.Time, endDate *time.Time, city, url *string,
) (*model.UserCompetition, error) {
	const query = `
		insert into user_competition (user_id, competition_id, title, date, end_date, city, url)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id, user_id, competition_id, title, date, end_date, city, url, result, created_at, updated_at
	`

	c, err := scanUserCompetition(r.db.QueryRow(ctx, query, userID, competitionID, title, date, endDate, city, url))
	if err != nil {
		return nil, fmt.Errorf("create user competition from catalog: %w", err)
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

// ListCatalogUpcoming returns up to limit published catalog entries whose
// COALESCE(end_date, date) hasn't passed today, soonest first, starting at
// offset, plus whether more exist past this page. city nil means no filter
// (show everything); otherwise it's matched case-insensitively.
func (r *Repository) ListCatalogUpcoming(
	ctx context.Context, today time.Time, city *string, limit, offset int,
) ([]*model.Competition, bool, error) {
	const query = `
		select id, title, date, end_date, city, status, created_by, created_at
		from competition
		where status = $1
		  and coalesce(end_date, date) >= $2
		  and ($3::text is null or lower(city) = lower($3))
		order by date asc, id asc
		limit $4 offset $5
	`

	rows, err := r.db.Query(ctx, query, competitionStatusToDB(model.CompetitionStatusPublished), today, city, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list catalog upcoming: %w", err)
	}
	defer rows.Close()

	competitions, err := scanCompetitions(rows)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(competitions) > limit
	if hasMore {
		competitions = competitions[:limit]
	}

	return competitions, hasMore, nil
}

// HasPublishedUpcoming reports whether at least one published catalog
// entry hasn't passed today yet — competition/list uses this to decide
// whether the "🔎 Найти" entry point is even worth showing, without
// paying for a full list it would just discard.
func (r *Repository) HasPublishedUpcoming(ctx context.Context, today time.Time) (bool, error) {
	const query = `
		select exists(
			select 1
			from competition
			where status = $1 and coalesce(end_date, date) >= $2
		)
	`

	var exists bool

	err := r.db.QueryRow(ctx, query, competitionStatusToDB(model.CompetitionStatusPublished), today).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check for published upcoming competitions: %w", err)
	}

	return exists, nil
}

func (r *Repository) GetCompetition(ctx context.Context, id int64) (*model.Competition, error) {
	const query = `
		select id, title, date, end_date, city, status, created_by, created_at
		from competition
		where id = $1
	`

	c, err := scanCompetition(r.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get competition by id: %w", err)
	}

	return c, nil
}

// GetPrimarySourceURL returns the organizer-page URL marked primary for a
// competition, if one exists — nil (not an error) if the competition has
// no source recorded as primary.
func (r *Repository) GetPrimarySourceURL(ctx context.Context, competitionID int64) (*string, error) {
	const query = `select url from competition_source where competition_id = $1 and is_primary = true`

	var url string

	err := r.db.QueryRow(ctx, query, competitionID).Scan(&url)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get primary competition source: %w", err)
	}

	return &url, nil
}

// FindSourceByURL looks up which competition (if any) a normalized URL is
// already a known source of — the dedup check competition/submit runs
// before ever creating a new catalog entry.
func (r *Repository) FindSourceByURL(ctx context.Context, url string) (*model.CompetitionSource, error) {
	const query = `select competition_id, url, is_primary from competition_source where url = $1`

	s, err := scanCompetitionSource(r.db.QueryRow(ctx, query, url))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find competition source by url: %w", err)
	}

	return s, nil
}

// FindCandidates returns pending or published competitions within 2 days
// of date in the same city (case-insensitive) — the "Это один из них?"
// list competition/submit shows before ever creating a new catalog entry.
func (r *Repository) FindCandidates(ctx context.Context, date time.Time, city string) ([]*model.Competition, error) {
	const query = `
		select id, title, date, end_date, city, status, created_by, created_at
		from competition
		where status in ($1, $2)
		  and date between $3::date - interval '2 days' and $3::date + interval '2 days'
		  and lower(city) = lower($4)
		order by date asc, id asc
	`

	rows, err := r.db.Query(
		ctx, query, competitionStatusToDB(model.CompetitionStatusPending), competitionStatusToDB(model.CompetitionStatusPublished),
		date, city,
	)
	if err != nil {
		return nil, fmt.Errorf("find candidate competitions: %w", err)
	}
	defer rows.Close()

	return scanCompetitions(rows)
}

// SearchCompetitionsByTitle is the admin's own duplicate-picker — a plain
// substring search, since by the time "🔗 Это дубль" is tapped the
// automatic FindCandidates search (same date/city heuristic) has already
// missed whatever the admin is about to find manually.
func (r *Repository) SearchCompetitionsByTitle(
	ctx context.Context, term string, excludeID int64, limit int,
) ([]*model.Competition, error) {
	const query = `
		select id, title, date, end_date, city, status, created_by, created_at
		from competition
		where id != $1 and title ilike '%' || $2 || '%'
		order by date desc, id desc
		limit $3
	`

	rows, err := r.db.Query(ctx, query, excludeID, term, limit)
	if err != nil {
		return nil, fmt.Errorf("search competitions by title: %w", err)
	}
	defer rows.Close()

	return scanCompetitions(rows)
}

func (r *Repository) CreateCompetition(
	ctx context.Context, title string, date time.Time, endDate *time.Time, city *string, createdBy int64,
) (*model.Competition, error) {
	const query = `
		insert into competition (title, date, end_date, city, status, created_by)
		values ($1, $2, $3, $4, $5, $6)
		returning id, title, date, end_date, city, status, created_by, created_at
	`

	c, err := scanCompetition(
		r.db.QueryRow(ctx, query, title, date, endDate, city, competitionStatusToDB(model.CompetitionStatusPending), createdBy),
	)
	if err != nil {
		return nil, fmt.Errorf("create competition: %w", err)
	}

	return c, nil
}

func (r *Repository) AddCompetitionSource(ctx context.Context, competitionID int64, url string, isPrimary bool) error {
	const query = `
		insert into competition_source (competition_id, url, is_primary)
		values ($1, $2, $3)
	`

	_, err := r.db.Exec(ctx, query, competitionID, url, isPrimary)
	if err != nil {
		return fmt.Errorf("add competition source: %w", err)
	}

	return nil
}

// LinkUserCompetition points an existing personal record at a catalog
// entry — used both when attaching to an existing competition (exact URL
// or candidate match) and right after creating a brand new pending one.
func (r *Repository) LinkUserCompetition(ctx context.Context, userCompetitionID, competitionID int64) error {
	const query = `update user_competition set competition_id = $2, updated_at = now() where id = $1`

	_, err := r.db.Exec(ctx, query, userCompetitionID, competitionID)
	if err != nil {
		return fmt.Errorf("link user competition: %w", err)
	}

	return nil
}

// ListOwnersByCompetitionID returns the distinct user_ids of every personal
// record currently linked to a competition — more than one submitter can
// end up attached to the same pending entry before it's moderated, and all
// of them need notifying once it is.
func (r *Repository) ListOwnersByCompetitionID(ctx context.Context, competitionID int64) ([]int64, error) {
	const query = `select distinct user_id from user_competition where competition_id = $1`

	rows, err := r.db.Query(ctx, query, competitionID)
	if err != nil {
		return nil, fmt.Errorf("list competition owners: %w", err)
	}
	defer rows.Close()

	var owners []int64

	for rows.Next() {
		var userID int64

		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan competition owner: %w", err)
		}

		owners = append(owners, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list competition owners: %w", err)
	}

	return owners, nil
}

// SetCompetitionStatus transitions a competition out of "pending" — guarded
// by WHERE status = pending so a second tap of ✅/❌ on an
// already-moderated entry (Telegram can deliver a callback twice) is a
// silent no-op rather than a double notification. updated reports whether
// this call was the one that actually applied the change.
func (r *Repository) SetCompetitionStatus(ctx context.Context, id int64, status model.CompetitionStatus) (bool, error) {
	const query = `update competition set status = $2 where id = $1 and status = $3`

	tag, err := r.db.Exec(ctx, query, id, competitionStatusToDB(status), competitionStatusToDB(model.CompetitionStatusPending))
	if err != nil {
		return false, fmt.Errorf("set competition status: %w", err)
	}

	return tag.RowsAffected() > 0, nil
}

// MergeCompetition folds a pending submission into an existing competition
// the admin picked via "🔗 Это дубль": every competition_source row
// pointing at the pending entry gets repointed and demoted to non-primary
// (the existing entry already has its own primary source — the partial
// unique index would reject a second one), every user_competition row
// gets repointed the same way, and the now-empty pending row is deleted —
// guarded by status = pending so a second tap after it's already been
// merged (or approved/rejected from elsewhere) is a silent no-op. All
// three statements run in one transaction: a crash between them must never
// leave a competition_source or user_competition row pointing at a
// competition that no longer exists.
func (r *Repository) MergeCompetition(ctx context.Context, pendingID, existingID int64) (bool, error) {
	var merged bool

	err := tx.WithTx(ctx, r.db, func(ctx context.Context) error {
		q := tx.QuerierFromContext(ctx, r.db)

		_, err := q.Exec(
			ctx, `update competition_source set competition_id = $2, is_primary = false where competition_id = $1`,
			pendingID, existingID,
		)
		if err != nil {
			return fmt.Errorf("repoint competition sources: %w", err)
		}

		_, err = q.Exec(
			ctx, `update user_competition set competition_id = $2, updated_at = now() where competition_id = $1`,
			pendingID, existingID,
		)
		if err != nil {
			return fmt.Errorf("repoint user competitions: %w", err)
		}

		tag, err := q.Exec(
			ctx, `delete from competition where id = $1 and status = $2`,
			pendingID, competitionStatusToDB(model.CompetitionStatusPending),
		)
		if err != nil {
			return fmt.Errorf("delete pending competition: %w", err)
		}

		merged = tag.RowsAffected() > 0

		return nil
	})
	if err != nil {
		return false, err
	}

	return merged, nil
}

func (r *Repository) GetMergeDraftByUserID(ctx context.Context, userID int64) (*model.CompetitionMergeDraft, error) {
	const query = `
		select user_id, pending_competition_id, created_at
		from competition_merge_draft
		where user_id = $1
	`

	d, err := scanMergeDraft(r.db.QueryRow(ctx, query, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get competition merge draft by user id: %w", err)
	}

	return d, nil
}

func (r *Repository) SetMergeDraft(ctx context.Context, userID, pendingCompetitionID int64) error {
	const query = `
		insert into competition_merge_draft (user_id, pending_competition_id)
		values ($1, $2)
		on conflict (user_id) do update set pending_competition_id = $2, created_at = now()
	`

	_, err := r.db.Exec(ctx, query, userID, pendingCompetitionID)
	if err != nil {
		return fmt.Errorf("set competition merge draft: %w", err)
	}

	return nil
}

func (r *Repository) DeleteMergeDraft(ctx context.Context, userID int64) error {
	const query = `delete from competition_merge_draft where user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete competition merge draft: %w", err)
	}

	return nil
}

// GetViewFilter looks up the catalog browse screen's active city override
// for userID, if any. ErrNotFound means "no override" — callers should
// fall back to the user's own City field, not treat it as an error.
func (r *Repository) GetViewFilter(ctx context.Context, userID int64) (*model.CatalogViewFilter, error) {
	const query = `select user_id, city, updated_at from catalog_view_filter where user_id = $1`

	f, err := scanViewFilter(r.db.QueryRow(ctx, query, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get catalog view filter by user id: %w", err)
	}

	return f, nil
}

func (r *Repository) SetViewFilter(ctx context.Context, userID int64, city *string) error {
	const query = `
		insert into catalog_view_filter (user_id, city)
		values ($1, $2)
		on conflict (user_id) do update set city = $2, updated_at = now()
	`

	_, err := r.db.Exec(ctx, query, userID, city)
	if err != nil {
		return fmt.Errorf("set catalog view filter: %w", err)
	}

	return nil
}

func (r *Repository) GetCityDraftByUserID(ctx context.Context, userID int64) (*model.CatalogCityDraft, error) {
	const query = `select user_id, created_at from catalog_city_draft where user_id = $1`

	d, err := scanCityDraft(r.db.QueryRow(ctx, query, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, model.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get catalog city draft by user id: %w", err)
	}

	return d, nil
}

func (r *Repository) SetCityDraft(ctx context.Context, userID int64) error {
	const query = `
		insert into catalog_city_draft (user_id)
		values ($1)
		on conflict (user_id) do update set created_at = now()
	`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("set catalog city draft: %w", err)
	}

	return nil
}

func (r *Repository) DeleteCityDraft(ctx context.Context, userID int64) error {
	const query = `delete from catalog_city_draft where user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete catalog city draft: %w", err)
	}

	return nil
}

func scanViewFilter(row pgx.Row) (*model.CatalogViewFilter, error) {
	var f model.CatalogViewFilter

	err := row.Scan(&f.UserID, &f.City, &f.UpdatedAt)
	if err != nil {
		return nil, err
	}

	return &f, nil
}

func scanCityDraft(row pgx.Row) (*model.CatalogCityDraft, error) {
	var d model.CatalogCityDraft

	err := row.Scan(&d.UserID, &d.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func scanMergeDraft(row pgx.Row) (*model.CompetitionMergeDraft, error) {
	var d model.CompetitionMergeDraft

	err := row.Scan(&d.UserID, &d.PendingCompetitionID, &d.CreatedAt)
	if err != nil {
		return nil, err
	}

	return &d, nil
}

func scanCompetition(row pgx.Row) (*model.Competition, error) {
	var (
		c      model.Competition
		status int16
	)

	err := row.Scan(&c.ID, &c.Title, &c.Date, &c.EndDate, &c.City, &status, &c.CreatedBy, &c.CreatedAt)
	if err != nil {
		return nil, err
	}

	c.Status = competitionStatusFromDB(status)

	return &c, nil
}

func scanCompetitions(rows pgx.Rows) ([]*model.Competition, error) {
	var competitions []*model.Competition

	for rows.Next() {
		c, err := scanCompetition(rows)
		if err != nil {
			return nil, fmt.Errorf("scan competition: %w", err)
		}

		competitions = append(competitions, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list competitions: %w", err)
	}

	return competitions, nil
}

func scanCompetitionSource(row pgx.Row) (*model.CompetitionSource, error) {
	var s model.CompetitionSource

	err := row.Scan(&s.CompetitionID, &s.URL, &s.IsPrimary)
	if err != nil {
		return nil, err
	}

	return &s, nil
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
