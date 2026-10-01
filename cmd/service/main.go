package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/khambek-archakov/jitLog/internal/configure"
	tgbotapigateway "github.com/khambek-archakov/jitLog/internal/gateway/tgbotapi"
	updatehandler "github.com/khambek-archakov/jitLog/internal/handler/update"
	trainingrepo "github.com/khambek-archakov/jitLog/internal/repository/training"
	userrepo "github.com/khambek-archakov/jitLog/internal/repository/user"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding"
	"github.com/khambek-archakov/jitLog/internal/usecase/router"
	"github.com/khambek-archakov/jitLog/internal/usecase/router/chain"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/create"
	trainingdelete "github.com/khambek-archakov/jitLog/internal/usecase/training/delete"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/history"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/stats"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/update"
)

const (
	success = 0
	fail    = 1
)

func main() {
	os.Exit(run())
}

func run() int {
	// load env
	_ = godotenv.Load()

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		slog.Error("TELEGRAM_TOKEN is not set")
		return fail
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		slog.Error("DATABASE_URL is not set")
		return fail
	}

	// logger
	logger := slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}),
	)
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer cancel()

	// postgres
	DB, err := configure.InitDB(ctx, databaseURL)
	if err != nil {
		logger.Error("failed to configure postgres", "error", err)
		return fail
	}
	defer DB.Close()

	logger.Info("postgres connected")

	// telegram bot
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		logger.Error("failed to init telegram bot", "error", err)
		return fail
	}

	logger.Info("telegram bot started", "username", bot.Self.UserName)

	users := userrepo.New(DB)
	trainings := trainingrepo.New(DB)
	gateway := tgbotapigateway.New(bot)

	onboardingUseCase := onboarding.New(gateway, users)
	trainingCreateUseCase := create.New(gateway, trainings)
	trainingInfoUseCase := info.New(gateway, trainings)
	trainingHistoryUseCase := history.New(gateway, trainings)
	trainingUpdateUseCase := update.New(gateway, trainings)
	trainingDeleteUseCase := trainingdelete.New(gateway, trainings)
	trainingStatsUseCase := stats.New(gateway, trainings)

	handlers := chain.New(
		onboardingUseCase, trainingCreateUseCase, trainingInfoUseCase, trainingHistoryUseCase, trainingUpdateUseCase,
		trainingDeleteUseCase, trainingStatsUseCase, trainings, trainings,
	).Default()
	appRouter := router.New(handlers, users)

	updateHandler := updatehandler.New(appRouter, logger)

	// prometheus + health server
	httpMux := http.NewServeMux()

	httpMux.Handle("/metrics", promhttp.Handler())

	httpMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           httpMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("http server started", "addr", httpServer.Addr)

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("http server failed", "error", err)
			cancel()
		}
	}()

	// telegram updates
	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 30

	updates := bot.GetUpdatesChan(updateConfig)

	go updateHandler.Handle(ctx, updates)

	<-ctx.Done()

	logger.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("http shutdown failed", "error", err)
		return fail
	}

	logger.Info("application stopped")

	return success
}
