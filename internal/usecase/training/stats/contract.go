//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package stats

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

type trainingRepo interface {
	// ListAllTrainings returns userID's full training history, oldest
	// first — stats needs it all, both for the period aggregates and for
	// the streak, which spans further back than any single period.
	ListAllTrainings(ctx context.Context, userID int64) ([]*model.Training, error)
}

type beltRepo interface {
	// ListBeltPromotions returns userID's belt history, oldest first —
	// used both to drive the "По поясам" breakdown and to decide whether
	// the mode switcher should show at all (only once there are ≥2).
	ListBeltPromotions(ctx context.Context, userID int64) ([]*model.BeltPromotion, error)
}
