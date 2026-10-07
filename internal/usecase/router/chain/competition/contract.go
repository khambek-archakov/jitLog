//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package competition

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// handler is one trigger in this domain's own internal sequence — see
// training's identical contract.go comment for why it's declared here
// rather than imported from the parent chain package.
type handler interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// competitionCreate is what this domain needs from the "add a tournament"
// dialog — just enough to kick a fresh draft off. Unlike
// training/schedule's own Begin, this takes the full *model.User (not a
// bare userID) because the wizard's terminal step needs u.Timezone to
// render the confirmation card's status line.
type competitionCreate interface {
	Begin(ctx context.Context, u *model.User, in dto.Input) error
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
// (competition:edit:*). This package only ever drives it by callback,
// never Continue — that's the parent chain package's own
// competitionEditDraftContinuation.
type competitionUpdate interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// competitionDelete owns competition:delete:* (a confirm screen, then the
// actual delete).
type competitionDelete interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// competitionSubmit owns "📤 Предложить в каталог" (competition:submit:*).
type competitionSubmit interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}

// competitionModerate owns the admin's ✅/❌/🔗 actions
// (competition:moderate:*). This package only ever drives it by callback,
// never Continue — that's the parent chain package's own
// competitionMergeDraftContinuation.
type competitionModerate interface {
	Handle(ctx context.Context, u *model.User, in dto.Input) error
}
