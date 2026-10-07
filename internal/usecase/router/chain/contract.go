//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package chain

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// onboarding is what the chain needs from the onboarding scenario — it
// knows nothing about training, it just handles the input for an
// already-known user.
type onboarding interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// trainingCreate is what the chain needs from the "add a training" dialog.
// Begin starts a fresh draft; Continue dispatches into an existing one.
type trainingCreate interface {
	Begin(ctx context.Context, userID int64, in dto.Input) error
	Continue(ctx context.Context, d *model.TrainingDraft, in dto.Input) error
}

// trainingInfo shows a single training's card (training:view:{id}).
type trainingInfo interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// trainingHistory shows the paginated list of logged trainings
// (training:history:page:{n}).
type trainingHistory interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// trainingUpdate edits a single field of an existing training
// (training:edit:*). Handle covers every callback-driven step; Continue
// only fires for the two fields whose value arrives as free text
// (duration's "Другое" and notes), once a pending edit draft exists.
type trainingUpdate interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
	Continue(ctx context.Context, d *model.TrainingEditDraft, in dto.Input) error
}

// trainingDelete owns training:delete:* (a confirm screen, then the
// actual delete) — also fully stateless, same as most of trainingUpdate.
type trainingDelete interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// trainingStats shows the "📊 Статистика" screen (stats:period:*).
type trainingStats interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// profile shows the "👤 Профиль" screen and its belt-change/age-change
// flows (profile:*). Needs the full *model.User, same as onboarding — it
// reads and mutates it directly, not just the ID. Continue only fires for
// age's free-text reply, once a pending profile edit draft exists.
type profile interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
	Continue(ctx context.Context, u *model.User, d *model.ProfileEditDraft, in dto.Input) error
}

// scheduleCreate is what the chain needs from the "add a schedule slot"
// dialog — same Begin/Continue shape as trainingCreate.
type scheduleCreate interface {
	Begin(ctx context.Context, userID int64, in dto.Input) error
	Continue(ctx context.Context, d *model.ScheduleDraft, in dto.Input) error
}

// scheduleList shows the "📅 Расписание" screen (schedule:list).
type scheduleList interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// scheduleInfo shows a single slot's card (schedule:view:{id}).
type scheduleInfo interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

// scheduleUpdate edits a single field of an existing slot
// (schedule:edit:*). Handle covers every callback-driven step; Continue
// only fires for time's "Другое" fallback, once a pending edit draft
// exists.
type scheduleUpdate interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
	Continue(ctx context.Context, d *model.ScheduleEditDraft, in dto.Input) error
}

// scheduleDelete owns schedule:delete:* (a confirm screen, then the
// actual delete) — also fully stateless, same as most of scheduleUpdate.
type scheduleDelete interface {
	Handle(ctx context.Context, userID int64, in dto.Input) error
}

type trainingDraft interface {
	GetDraftByUserID(ctx context.Context, userID int64) (*model.TrainingDraft, error)
}

type trainingEditDraft interface {
	GetEditDraftByUserID(ctx context.Context, userID int64) (*model.TrainingEditDraft, error)
}

type scheduleDraft interface {
	GetDraftByUserID(ctx context.Context, userID int64) (*model.ScheduleDraft, error)
}

type scheduleEditDraft interface {
	GetEditDraftByUserID(ctx context.Context, userID int64) (*model.ScheduleEditDraft, error)
}

type profileEditDraft interface {
	GetEditDraftByUserID(ctx context.Context, userID int64) (*model.ProfileEditDraft, error)
}

// competitionCreate is what the chain needs from the "add a tournament"
// dialog — same Begin/Continue shape as trainingCreate/scheduleCreate,
// except both take the full *model.User (not a bare userID): the wizard's
// terminal step needs u.Timezone to render the confirmation card.
type competitionCreate interface {
	Begin(ctx context.Context, u *model.User, in dto.Input) error
	Continue(ctx context.Context, u *model.User, d *model.UserCompetitionDraft, in dto.Input) error
}

// competitionList shows the "🏆 Соревнования" screen (competition:list).
type competitionList interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// competitionHistory shows the paginated past-tournaments screen
// (competition:history:page:{n}).
type competitionHistory interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// competitionInfo shows a single tournament's card (competition:view:{id}).
type competitionInfo interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// competitionUpdate edits a single field of an existing tournament
// (competition:edit:*). Handle covers every callback-driven step;
// Continue fires for all six fields, once a pending edit draft exists —
// unlike training/schedule, nothing here is pickable from a button.
type competitionUpdate interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
	Continue(ctx context.Context, u *model.User, d *model.UserCompetitionEditDraft, in dto.Input) error
}

// competitionDelete owns competition:delete:* (a confirm screen, then the
// actual delete) — also fully stateless, same as most of competitionUpdate.
type competitionDelete interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

type competitionDraft interface {
	GetDraftByUserID(ctx context.Context, userID int64) (*model.UserCompetitionDraft, error)
}

type competitionEditDraft interface {
	GetEditDraftByUserID(ctx context.Context, userID int64) (*model.UserCompetitionEditDraft, error)
}

// competitionSubmit owns "📤 Предложить в каталог" (competition:submit:*).
type competitionSubmit interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// competitionModerate owns the admin's ✅/❌/🔗 actions on a pending
// submission (competition:moderate:*). Continue fires for the merge
// draft's own free-text title search, once a pending one exists.
type competitionModerate interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
	Continue(ctx context.Context, u *model.User, d *model.CompetitionMergeDraft, in dto.Input) error
}

type competitionMergeDraft interface {
	GetMergeDraftByUserID(ctx context.Context, userID int64) (*model.CompetitionMergeDraft, error)
}

// catalogList shows the "📚 Каталог" screen (catalog:list, its pagination
// and its city-filter controls). Continue fires for the city draft's own
// free-text city name, once a pending one exists.
type catalogList interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
	Continue(ctx context.Context, u *model.User, d *model.CatalogCityDraft, in dto.Input) error
}

// catalogInfo shows a single catalog entry's card (catalog:view:{id}).
type catalogInfo interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// catalogAdd owns "➕ В мои" (catalog:add:{id}).
type catalogAdd interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

type catalogCityDraft interface {
	GetCityDraftByUserID(ctx context.Context, userID int64) (*model.CatalogCityDraft, error)
}
