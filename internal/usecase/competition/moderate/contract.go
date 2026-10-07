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
}

// userRepo resolves a user_id (all we have from
// ListOwnersByCompetitionID) to a full user row — just enough to get at
// TelegramID for a notification.
type userRepo interface {
	GetByID(ctx context.Context, id int64) (*model.User, error)
}
