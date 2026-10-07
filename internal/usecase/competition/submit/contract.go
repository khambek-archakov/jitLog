//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package submit

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

type repo interface {
	GetUserCompetition(ctx context.Context, id int64) (*model.UserCompetition, error)
	FindSourceByURL(ctx context.Context, url string) (*model.CompetitionSource, error)
	FindCandidates(ctx context.Context, date time.Time, city string) ([]*model.Competition, error)
	CreateCompetition(
		ctx context.Context, title string, date time.Time, endDate *time.Time, city *string, createdBy int64,
	) (*model.Competition, error)
	AddCompetitionSource(ctx context.Context, competitionID int64, url string, isPrimary bool) error
	LinkUserCompetition(ctx context.Context, userCompetitionID, competitionID int64) error
}
