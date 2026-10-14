// Command ajp runs one AJP competition import (see
// internal/usecase/competition/import/ajp) and exits — it holds no
// business logic of its own, only wiring: load config, acquire the
// cross-process advisory lock, build the usecase's dependencies, call it
// once, report the outcome. Scheduling is systemd's job (see
// deploy/systemd), not this binary's.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"github.com/khambek-archakov/jitLog/internal/configure"
	tgbotapigateway "github.com/khambek-archakov/jitLog/internal/gateway/tgbotapi"
	competitionrepo "github.com/khambek-archakov/jitLog/internal/repository/competition"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/import/ajp"
)

const (
	success = 0
	fail    = 1
)

// defaultAJPCountries is AJP_COUNTRIES' value when the env var isn't set
// at all — explicitly setting it to an empty string is a different,
// deliberate choice (see the usecase's own New/Run) and must not fall
// back to this.
var defaultAJPCountries = []string{"RU", "BY", "AZ", "AM", "KZ"}

const defaultCityDictionaryPath = "configs/ajp_cities.txt"

// advisoryLockKey is an arbitrary fixed id, unique to this one cron job —
// pg_try_advisory_lock's keyspace is shared process-wide, so every lock
// user in this project needs its own distinct constant.
const advisoryLockKey = 725010001

const requestTimeout = 15 * time.Second

func main() {
	os.Exit(run())
}

func run() int {
	_ = godotenv.Load()

	dryRun := flag.Bool("dry-run", false, "parse and compare only, write nothing and send no Telegram messages")
	fromFile := flag.String(
		"from-file", "", "read a previously saved copy of the page instead of fetching it live (see deploy/systemd/README.md)",
	)
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		logger.Error("TELEGRAM_TOKEN is not set")
		return fail
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		logger.Error("DATABASE_URL is not set")
		return fail
	}

	adminTelegramIDRaw := os.Getenv("ADMIN_TELEGRAM_ID")
	if adminTelegramIDRaw == "" {
		logger.Error("ADMIN_TELEGRAM_ID is not set")
		return fail
	}

	adminTelegramID, err := strconv.ParseInt(adminTelegramIDRaw, 10, 64)
	if err != nil {
		logger.Error("ADMIN_TELEGRAM_ID is not a valid integer", "error", err)
		return fail
	}

	countries := defaultAJPCountries
	if raw, ok := os.LookupEnv("AJP_COUNTRIES"); ok {
		countries = splitCountries(raw)
	}

	cityDictionaryPath := defaultCityDictionaryPath
	if raw := os.Getenv("AJP_CITY_DICTIONARY_PATH"); raw != "" {
		cityDictionaryPath = raw
	}

	cityDict, err := ajp.LoadCityDictionary(cityDictionaryPath)
	if err != nil {
		logger.Error("failed to load city dictionary", "path", cityDictionaryPath, "error", err)
		return fail
	}

	ctx := context.Background()

	db, err := configure.InitDB(ctx, databaseURL)
	if err != nil {
		logger.Error("failed to configure postgres", "error", err)
		return fail
	}
	defer db.Close()

	conn, err := db.Acquire(ctx)
	if err != nil {
		logger.Error("failed to acquire a dedicated connection for the advisory lock", "error", err)
		return fail
	}
	defer conn.Release()

	var locked bool

	if err := conn.QueryRow(ctx, "select pg_try_advisory_lock($1)", advisoryLockKey).Scan(&locked); err != nil {
		logger.Error("failed to acquire advisory lock", "error", err)
		return fail
	}

	if !locked {
		logger.Info("another AJP import run is already in progress, exiting")
		return success
	}

	defer func() {
		if _, err := conn.Exec(context.Background(), "select pg_advisory_unlock($1)", advisoryLockKey); err != nil {
			logger.Error("failed to release advisory lock", "error", err)
		}
	}()

	bot, err := tgbotapi.NewBotAPIWithClient(token, tgbotapi.APIEndpoint, &http.Client{Timeout: 45 * time.Second})
	if err != nil {
		logger.Error("failed to init telegram bot", "error", err)
		return fail
	}

	gateway := tgbotapigateway.New(bot)
	competitions := competitionrepo.New(db)

	var importer *ajp.UseCase

	if *fromFile != "" {
		// A human already solved whatever bot-check the live site might
		// show by saving this copy in their own browser — QuietFailureSender
		// just keeps the one monitoring-failure message from also hitting
		// Telegram, since that same person is watching this terminal.
		importer = ajp.New(ajp.QuietFailureSender(gateway), competitions, ajp.NewFilePageLoader(*fromFile),
			adminTelegramID, countries, cityDict)
		importer.SetSourceLabel("файл " + filepath.Base(*fromFile))
	} else {
		importer = ajp.New(gateway, competitions, ajp.NewHTTPPageLoader(requestTimeout), adminTelegramID, countries, cityDict)
	}

	report, err := importer.Run(ctx, *dryRun)
	if err != nil {
		logger.Error("ajp import failed", "error", err)
		return fail
	}

	logger.Info("ajp import finished", "report", ajp.FormatReport(report))

	if *dryRun {
		fmt.Println(ajp.FormatReport(report))
	}

	return success
}

func splitCountries(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return []string{}
	}

	parts := strings.Split(raw, ",")
	countries := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			countries = append(countries, p)
		}
	}

	return countries
}
