// Package ajp owns one daily import run: fetch AJP's own federation
// events page, turn it into pending catalog entries nobody has to type in
// by hand, and leave anything uncertain for the admin to decide via
// competition/moderate (see that package's own "ajp:..." callbacks).
//
// It never fetches a single event's own page (/en/event/{id}) — that's
// bot-protected on the live site and explicitly off-limits — everything
// it needs rides the one list page's own data-* attributes and the city
// line next to each card.
package ajp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/import/ajp/parse"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/urlnorm"
)

// sourceURL is the one page this importer is ever allowed to read — never
// configurable, so a bad env var can't quietly point it somewhere else.
const sourceURL = "https://ajptour.com/en/federation/1/events"

// userAgent identifies this bot honestly rather than pretending to be a
// browser — the site's own robots.txt allows this path for User-agent: *.
const userAgent = "JitLogBot (+https://t.me/JitLogBot)"

// maxCombinedMessageItems is the cutoff between "👀 Смотреть по одному"
// (used on the first-ever run, or once a run creates more than this many)
// and one message holding every new competition's own moderation buttons.
const maxCombinedMessageItems = 10

// Report summarizes exactly one Run — what it found, what it did (or
// would have done, under --dry-run), and anything that needs a human's
// attention. Nothing here depends on whether the run actually wrote to
// the database, so the same type and the same FormatReport serve both a
// real run's log line and --dry-run's own stdout printout.
type Report struct {
	TotalCards    int
	CountryCounts map[string]int

	// Created holds, for a real run, the row exactly as the database
	// returned it; for --dry-run, a Competition built in memory purely
	// for display (its ID is always zero — nothing was inserted).
	Created []*model.Competition
	Diffs   []DiffNotice
	Skipped []SkippedCard

	// UntranslatedCities lists, once each, every English city name this
	// run saw that configs/ajp_cities.txt has no entry for — the city
	// itself was still imported as-is, this is purely a nudge to extend
	// the dictionary.
	UntranslatedCities []string

	// FirstRun is true when no pending, author-less competition existed
	// yet before this run — it and ">10 created" both switch the new-
	// competitions notification from one-message-per-batch to the
	// "✅ Одобрить все" / "👀 Смотреть по одному" summary.
	FirstRun bool
}

// DiffNotice is one existing competition whose date or city the source
// now disagrees with — never applied automatically, only ever proposed
// (see model.CompetitionImportDiff).
type DiffNotice struct {
	Competition     *model.Competition
	ProposedDate    time.Time
	ProposedEndDate *time.Time
	ProposedCity    string
}

// SkippedCard is one eventItem card this run chose not to import at all —
// an unparseable date, a date too far ahead, or a city it couldn't make
// sense of. A card outside AJP_COUNTRIES is never recorded here at all;
// decision 5 (see the main project memory) is explicit that those are
// ignored silently, not reported.
type SkippedCard struct {
	Name   string
	Reason string
}

type UseCase struct {
	bot         sender
	repo        repo
	pages       pageLoader
	adminChatID int64
	countries   map[string]struct{}
	cityDict    map[string]string
	now         func() time.Time
}

// New builds the importer. countries is AJP_COUNTRIES already split on
// comma — an empty (but non-nil) slice means "report only, import
// nothing" (decision 5), which is different from the caller choosing its
// own default list when the env var was unset entirely; New itself treats
// both the same way (whatever it's handed is exactly what's enforced).
func New(bot sender, r repo, pages pageLoader, adminChatID int64, countries []string, cityDict map[string]string) *UseCase {
	set := make(map[string]struct{}, len(countries))
	for _, c := range countries {
		set[strings.ToUpper(strings.TrimSpace(c))] = struct{}{}
	}

	return &UseCase{
		bot: bot, repo: r, pages: pages, adminChatID: adminChatID, countries: set, cityDict: cityDict, now: time.Now,
	}
}

// NewWithClock is New with an overridable clock — exported only so tests
// can pin "today" deterministically; every real caller just uses New.
func NewWithClock(
	bot sender, r repo, pages pageLoader, adminChatID int64, countries []string, cityDict map[string]string,
	now func() time.Time,
) *UseCase {
	uc := New(bot, r, pages, adminChatID, countries, cityDict)
	uc.now = now

	return uc
}

// Run performs one import. dryRun skips every database write and every
// Telegram send except the one monitoring-failure message (there is
// nothing to preview if the page itself never came back) — everything it
// would otherwise have done instead fills the returned Report.
func (uc *UseCase) Run(ctx context.Context, dryRun bool) (*Report, error) {
	body, status, err := uc.pages.Load(ctx)
	if err != nil {
		return nil, uc.monitoringFailure(ctx, fmt.Sprintf("страница недоступна: %v", err))
	}

	if status != 200 {
		return nil, uc.monitoringFailure(ctx, fmt.Sprintf("у страницы статус %d", status))
	}

	events, err := parse.ParseEvents(body)

	switch {
	case errors.Is(err, parse.ErrNoEvents):
		return nil, uc.monitoringFailure(ctx, "на странице нет ни одной карточки соревнования")

	case errors.Is(err, parse.ErrMissingAttributes):
		return nil, uc.monitoringFailure(ctx, "у карточек на странице не хватает нужных полей — похоже, вёрстка страницы изменилась")

	case err != nil:
		return nil, uc.monitoringFailure(ctx, fmt.Sprintf("страница не разбирается: %v", err))
	}

	today := startOfDayUTC(uc.now())

	report := &Report{TotalCards: len(events), CountryCounts: countByCountry(events)}

	if len(uc.countries) == 0 {
		return report, nil
	}

	pendingBefore, err := uc.repo.ListPendingUnauthored(ctx)
	if err != nil {
		return nil, fmt.Errorf("list pending unauthored competitions: %w", err)
	}

	report.FirstRun = len(pendingBefore) == 0

	seenUntranslated := make(map[string]bool)

	for _, event := range events {
		if _, ok := uc.countries[strings.ToUpper(event.CountryCode)]; !ok {
			continue
		}

		if err := uc.processEvent(ctx, event, today, dryRun, report, seenUntranslated); err != nil {
			return nil, err
		}
	}

	if dryRun {
		return report, nil
	}

	if err := uc.notify(ctx, report); err != nil {
		return nil, err
	}

	return report, nil
}

func (uc *UseCase) processEvent(
	ctx context.Context, event parse.Event, today time.Time, dryRun bool, report *Report, seenUntranslated map[string]bool,
) error {
	if event.City == nil {
		report.Skipped = append(report.Skipped, SkippedCard{Name: event.Name, Reason: "город не получается определить"})
		return nil
	}

	date, endDate, err := parse.ResolveDate(event.DateRaw, today)
	if err != nil {
		report.Skipped = append(report.Skipped, SkippedCard{Name: event.Name, Reason: dateSkipReason(event.DateRaw, err)})
		return nil
	}

	normalizedURL, ok := urlnorm.Normalize(event.URL)
	if !ok {
		report.Skipped = append(report.Skipped, SkippedCard{Name: event.Name, Reason: "ссылка не получается разобрать"})
		return nil
	}

	russianCity, translated := translateCity(uc.cityDict, *event.City)
	if !translated && !seenUntranslated[*event.City] {
		seenUntranslated[*event.City] = true
		report.UntranslatedCities = append(report.UntranslatedCities, *event.City)
	}

	var endDatePtr *time.Time
	if !endDate.Equal(date) {
		endDatePtr = &endDate
	}

	source, err := uc.repo.FindSourceByURL(ctx, normalizedURL)

	switch {
	case errors.Is(err, model.ErrNotFound):
		return uc.handleNew(ctx, event.Name, date, endDatePtr, russianCity, normalizedURL, dryRun, report)

	case err == nil:
		return uc.handleExisting(ctx, source.CompetitionID, date, endDatePtr, russianCity, dryRun, report)

	default:
		return fmt.Errorf("find competition source by url: %w", err)
	}
}

func (uc *UseCase) handleNew(
	ctx context.Context, title string, date time.Time, endDate *time.Time, city, url string, dryRun bool,
	report *Report,
) error {
	if dryRun {
		report.Created = append(report.Created, &model.Competition{Title: title, Date: date, EndDate: endDate, City: &city})
		return nil
	}

	created, err := uc.repo.CreateImportedCompetition(ctx, title, date, endDate, &city)
	if err != nil {
		return fmt.Errorf("create imported competition: %w", err)
	}

	if err := uc.repo.AddCompetitionSource(ctx, created.ID, url, true); err != nil {
		return fmt.Errorf("add competition source: %w", err)
	}

	report.Created = append(report.Created, created)

	return nil
}

func (uc *UseCase) handleExisting(
	ctx context.Context, competitionID int64, date time.Time, endDate *time.Time, city string, dryRun bool, report *Report,
) error {
	existing, err := uc.repo.GetCompetition(ctx, competitionID)
	if err != nil {
		return fmt.Errorf("get competition: %w", err)
	}

	if !competitionDrifted(existing, date, endDate, city) {
		return nil
	}

	if !dryRun {
		if err := uc.repo.CreateImportDiff(ctx, existing.ID, date, endDate, &city); err != nil {
			return fmt.Errorf("create competition import diff: %w", err)
		}
	}

	report.Diffs = append(
		report.Diffs, DiffNotice{Competition: existing, ProposedDate: date, ProposedEndDate: endDate, ProposedCity: city},
	)

	return nil
}

func (uc *UseCase) monitoringFailure(ctx context.Context, reason string) error {
	if err := uc.bot.Send(ctx, uc.adminChatID, "Импорт AJP не работает: "+reason); err != nil {
		return err
	}

	return fmt.Errorf("ajp import: %s", reason)
}

// dateSkipReason turns a parse.ResolveDate failure into a plain-language
// reason for Report.Skipped — the sentinel errors stay in English for
// errors.Is, this is just how it reads to the admin.
func dateSkipReason(dateRaw string, err error) string {
	switch {
	case errors.Is(err, parse.ErrDateTooFarAhead):
		return fmt.Sprintf("дата %q слишком далеко вперёд (больше 18 месяцев)", dateRaw)
	default:
		return fmt.Sprintf("дата %q не получается разобрать", dateRaw)
	}
}

func competitionDrifted(existing *model.Competition, date time.Time, endDate *time.Time, city string) bool {
	if !existing.Date.Equal(date) {
		return true
	}

	if !datePtrEqual(existing.EndDate, endDate) {
		return true
	}

	return existing.City == nil || *existing.City != city
}

func datePtrEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	return a.Equal(*b)
}

func startOfDayUTC(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func countByCountry(events []parse.Event) map[string]int {
	counts := make(map[string]int)
	for _, e := range events {
		counts[e.CountryCode]++
	}

	return counts
}
