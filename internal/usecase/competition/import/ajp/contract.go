//go:generate mockgen -source=$GOFILE -destination=mock_${GOPACKAGE}_test.go -package=${GOPACKAGE}_test
package ajp

import (
	"context"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// pageLoader fetches the AJP events page — the only network dependency,
// kept behind an interface so the usecase itself never imports net/http
// and stays testable with a canned response.
type pageLoader interface {
	// Load returns the response body and HTTP status code. A non-nil
	// error means the request itself failed (timeout, DNS, ...) — both
	// that and a non-200 status count as "the page didn't come back" for
	// monitoring purposes.
	Load(ctx context.Context) (body string, status int, err error)
}

type repo interface {
	FindSourceByURL(ctx context.Context, url string) (*model.CompetitionSource, error)
	GetCompetition(ctx context.Context, id int64) (*model.Competition, error)
	CreateImportedCompetition(
		ctx context.Context, title string, date time.Time, endDate *time.Time, city *string,
	) (*model.Competition, error)
	AddCompetitionSource(ctx context.Context, competitionID int64, url string, isPrimary bool) error
	CreateImportDiff(
		ctx context.Context, competitionID int64, proposedDate time.Time, proposedEndDate *time.Time, proposedCity *string,
	) error
	ListPendingUnauthored(ctx context.Context) ([]*model.Competition, error)
}

type sender interface {
	Send(ctx context.Context, chatID int64, text string) error
	SendWithKeyboard(ctx context.Context, chatID int64, text string, keyboard dto.Keyboard) error
}
