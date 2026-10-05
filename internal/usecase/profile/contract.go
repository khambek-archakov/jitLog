//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package profile

import (
	"context"
	"time"

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

type user interface {
	Update(ctx context.Context, u *model.User) error
	// AddBeltPromotion mirrors internal/usecase/onboarding/steps' own
	// private contract — onboarding backfills the very first row,
	// profile adds every one after.
	AddBeltPromotion(ctx context.Context, userID int64, belt model.Belt, promotedAt time.Time) error
}

// profileEditDraft is the pending-age-edit draft — age is the only
// profile field whose new value can't ride a callback button.
type profileEditDraft interface {
	GetEditDraftByUserID(ctx context.Context, userID int64) (*model.ProfileEditDraft, error)
	SetEditDraft(ctx context.Context, userID int64) error
	DeleteEditDraft(ctx context.Context, userID int64) error
}
