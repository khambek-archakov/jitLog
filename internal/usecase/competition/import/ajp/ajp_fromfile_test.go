package ajp_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/import/ajp"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func newUseCaseFromFile(
	bot *Mocksender, r *Mockrepo, path string, countries []string, cityDict map[string]string,
) *ajp.UseCase {
	uc := ajp.NewWithClock(bot, r, ajp.NewFilePageLoader(path), adminChatID, countries, cityDict, func() time.Time { return fixedToday })
	uc.SetSourceLabel("файл " + filepath.Base(path))

	return uc
}

func writeTestFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "page.html")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

	return path
}

func TestRun_fromFile_sameResultAsNetwork(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	bot, repo := NewMocksender(ctrl), NewMockrepo(ctrl)

	repo.EXPECT().ListPendingUnauthored(gomock.Any()).Return(nil, nil)
	repo.EXPECT().FindSourceByURL(gomock.Any(), gomock.Any()).Return(nil, model.ErrNotFound).AnyTimes()

	uc := ajp.NewWithClock(
		bot, repo, ajp.NewFilePageLoader("parse/testdata/ajp_events.html"),
		adminChatID, []string{"RU"}, nil, func() time.Time { return fixedToday },
	)

	report, err := uc.Run(context.Background(), true)
	require.NoError(t, err)

	// Same saved page parse_test.go already checks card-by-card — here we
	// only need the page-level counts a network fetch of the identical
	// bytes would have produced too.
	assert.Equal(t, 54, report.TotalCards)
	assert.Equal(t, 7, report.CountryCounts["RU"])
}

func TestRun_fromFile_monitoringFailures(t *testing.T) {
	t.Run("file does not exist", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		bot, repo := NewMocksender(ctrl), NewMockrepo(ctrl)

		path := filepath.Join(t.TempDir(), "missing.html")

		// No bot.EXPECT().Send at all: QuietFailureSender must swallow it.
		quiet := ajp.QuietFailureSender(bot)
		uc := ajp.NewWithClock(quiet, repo, ajp.NewFilePageLoader(path), adminChatID, []string{"RU"}, nil, func() time.Time { return fixedToday })

		var err error

		stderr := captureStderr(t, func() {
			_, err = uc.Run(context.Background(), false)
		})

		assert.Error(t, err)
		assert.NotEmpty(t, stderr)
	})

	t.Run("empty file", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		bot, repo := NewMocksender(ctrl), NewMockrepo(ctrl)

		path := writeTestFile(t, "")

		quiet := ajp.QuietFailureSender(bot)
		uc := ajp.NewWithClock(quiet, repo, ajp.NewFilePageLoader(path), adminChatID, []string{"RU"}, nil, func() time.Time { return fixedToday })

		var err error

		stderr := captureStderr(t, func() {
			_, err = uc.Run(context.Background(), false)
		})

		assert.Error(t, err)
		assert.NotEmpty(t, stderr)
	})

	t.Run("no section#upcoming-events", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		bot, repo := NewMocksender(ctrl), NewMockrepo(ctrl)

		path := writeTestFile(t, `<html><body><p>nothing here</p></body></html>`)

		quiet := ajp.QuietFailureSender(bot)
		uc := ajp.NewWithClock(quiet, repo, ajp.NewFilePageLoader(path), adminChatID, []string{"RU"}, nil, func() time.Time { return fixedToday })

		var err error

		stderr := captureStderr(t, func() {
			_, err = uc.Run(context.Background(), false)
		})

		assert.Error(t, err)
		assert.NotEmpty(t, stderr)
	})

	t.Run("not a single eventItem card", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		bot, repo := NewMocksender(ctrl), NewMockrepo(ctrl)

		path := writeTestFile(t, pageHTML())

		quiet := ajp.QuietFailureSender(bot)
		uc := ajp.NewWithClock(quiet, repo, ajp.NewFilePageLoader(path), adminChatID, []string{"RU"}, nil, func() time.Time { return fixedToday })

		var err error

		stderr := captureStderr(t, func() {
			_, err = uc.Run(context.Background(), false)
		})

		assert.Error(t, err)
		assert.NotEmpty(t, stderr)
	})

	t.Run("a card missing data-url — same structural failure", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		bot, repo := NewMocksender(ctrl), NewMockrepo(ctrl)

		path := writeTestFile(t, `<html><body><section id="upcoming-events">
			<div class="event-bg cover-bg eventItem" data-id="1" data-name="X" data-date="October 10" data-country="RU"></div>
		</section></body></html>`)

		quiet := ajp.QuietFailureSender(bot)
		uc := ajp.NewWithClock(quiet, repo, ajp.NewFilePageLoader(path), adminChatID, []string{"RU"}, nil, func() time.Time { return fixedToday })

		var err error

		stderr := captureStderr(t, func() {
			_, err = uc.Run(context.Background(), false)
		})

		assert.Error(t, err)
		assert.NotEmpty(t, stderr)
	})
}

func TestRun_fromFile_repeatRunCreatesNoDuplicates(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	bot, repo := NewMocksender(ctrl), NewMockrepo(ctrl)

	path := writeTestFile(t, pageHTML(
		card(1, "Moscow Open", "October 10", "https://ajptour.com/en/event/1", "RU", "Venue, Moscow, Russia"),
	))

	repo.EXPECT().ListPendingUnauthored(gomock.Any()).Return(nil, nil)

	// Simulates running against the same file a second time, once the
	// first run's row is already in competition_source: FindSourceByURL
	// now succeeds, so no CreateImportedCompetition/AddCompetitionSource
	// call is expected at all.
	city := "Москва"
	repo.EXPECT().FindSourceByURL(gomock.Any(), "https://ajptour.com/en/event/1").
		Return(&model.CompetitionSource{CompetitionID: 50, URL: "https://ajptour.com/en/event/1", IsPrimary: true}, nil)
	repo.EXPECT().GetCompetition(gomock.Any(), int64(50)).
		Return(&model.Competition{
			ID: 50, Title: "Moscow Open", Date: time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC), City: &city,
		}, nil)

	uc := newUseCaseFromFile(bot, repo, path, []string{"RU"}, map[string]string{"Moscow": "Москва"})

	report, err := uc.Run(context.Background(), false)
	require.NoError(t, err)
	assert.Empty(t, report.Created)
	assert.Empty(t, report.Diffs)
}

func TestRun_fromFile_addsSourceLabelToNewCompetitionMessage(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	bot, repo := NewMocksender(ctrl), NewMockrepo(ctrl)

	path := writeTestFile(t, pageHTML(
		card(1, "Moscow Open", "October 10", "https://ajptour.com/en/event/1", "RU", "Venue, Moscow, Russia"),
	))

	repo.EXPECT().ListPendingUnauthored(gomock.Any()).Return([]*model.Competition{{ID: 999}}, nil)
	repo.EXPECT().FindSourceByURL(gomock.Any(), gomock.Any()).Return(nil, model.ErrNotFound)
	repo.EXPECT().
		CreateImportedCompetition(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(&model.Competition{ID: 7, Title: "Moscow Open"}, nil)
	repo.EXPECT().AddCompetitionSource(gomock.Any(), int64(7), gomock.Any(), true).Return(nil)

	bot.EXPECT().
		SendWithKeyboard(gomock.Any(), adminChatID, gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ int64, text string, _ dto.Keyboard) error {
			assert.Contains(t, text, "Источник: файл page.html")

			return nil
		})

	uc := newUseCaseFromFile(bot, repo, path, []string{"RU"}, map[string]string{"Moscow": "Москва"})

	_, err := uc.Run(context.Background(), false)
	require.NoError(t, err)
}
