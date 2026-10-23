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
	// ChangeBelt persists a belt change (and, when it's a genuine
	// promotion, the belt_promotion row backing it) atomically — see
	// internal/repository/user's own doc comment on it.
	ChangeBelt(ctx context.Context, userID int64, newBelt model.Belt, promotedAt time.Time, recordPromotion bool) (bool, error)
	// ListBeltPromotions finds the previous belt's own promoted_at, to
	// bound the "trainings on that belt" window in the congratulation
	// message.
	ListBeltPromotions(ctx context.Context, userID int64) ([]*model.BeltPromotion, error)
}

// trainingRepo is just enough to total up the previous belt's own
// trainings for the congratulation message's second line.
type trainingRepo interface {
	ListTrainingsBetween(ctx context.Context, userID int64, from, to time.Time) ([]*model.Training, error)
}

// profileEditDraft is the pending-age-edit draft — age is the only
// profile field whose new value can't ride a callback button.
type profileEditDraft interface {
	GetEditDraftByUserID(ctx context.Context, userID int64) (*model.ProfileEditDraft, error)
	SetEditDraft(ctx context.Context, userID int64) error
	DeleteEditDraft(ctx context.Context, userID int64) error
}
