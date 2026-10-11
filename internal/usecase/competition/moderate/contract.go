//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package moderate

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

type sender interface {
	Send(ctx context.Context, chatID int64, text string) error
	SendWithKeyboard(ctx context.Context, chatID int64, text string, keyboard dto.Keyboard) error
	EditMessageWithKeyboard(ctx context.Context, chatID int64, messageID int, text string, keyboard dto.Keyboard) error
	AnswerCallback(ctx context.Context, callbackID string) error
	AnswerCallbackWithText(ctx context.Context, callbackID, text string) error
}

type repo interface {
	SetCompetitionStatus(ctx context.Context, id int64, status model.CompetitionStatus) (bool, error)
	ListOwnersByCompetitionID(ctx context.Context, competitionID int64) ([]int64, error)
	SearchCompetitionsByTitle(ctx context.Context, term string, excludeID int64, limit int) ([]*model.Competition, error)
	MergeCompetition(ctx context.Context, pendingID, existingID int64) (bool, error)
	SetMergeDraft(ctx context.Context, userID, pendingCompetitionID int64) error
	DeleteMergeDraft(ctx context.Context, userID int64) error

	// ListPendingUnauthored backs "✅ Одобрить все" and "👀 Смотреть по
	// одному" — both act on every pending competition an automated import
	// (not a person) created, live at the moment they're tapped, not just
	// the batch a particular import run produced.
	ListPendingUnauthored(ctx context.Context) ([]*model.Competition, error)
	GetPrimarySourceURL(ctx context.Context, competitionID int64) (*string, error)
	ApplyImportDiff(ctx context.Context, competitionID int64) (bool, error)
	DeleteImportDiff(ctx context.Context, competitionID int64) error
}

// userRepo resolves a user_id (all we have from
// ListOwnersByCompetitionID) to a full user row — just enough to get at
// TelegramID for a notification.
type userRepo interface {
	GetByID(ctx context.Context, id int64) (*model.User, error)
}
