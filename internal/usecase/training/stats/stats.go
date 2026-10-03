// Package stats owns the "📊 Статистика" screen — read-only, no drafts.
// Two independent, non-combining views share one entry point: "За период"
// (stats:period:*, what the main menu's own button points at) and "По
// поясам" (stats:belts) — see the discussion behind not combining them:
// belt changes are rare enough that a period×belt filter mostly just
// reproduces one dimension or the other.
package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

type UseCase struct {
	bot   sender
	repo  trainingRepo
	belts beltRepo
}

func New(bot sender, repo trainingRepo, belts beltRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo, belts: belts}
}

func (uc *UseCase) Handle(ctx context.Context, userID int64, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	if in.CallbackData == callbackStatsBelts {
		return uc.handleBelts(ctx, userID, in)
	}

	p, ok := parsePeriod(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	return uc.handlePeriod(ctx, userID, in, p)
}

func (uc *UseCase) handlePeriod(ctx context.Context, userID int64, in dto.Input, p period) error {
	all, err := uc.repo.ListAllTrainings(ctx, userID)
	if err != nil {
		return fmt.Errorf("list all trainings: %w", err)
	}

	if len(all) == 0 {
		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, emptyStateText(), emptyStateKeyboard())
	}

	// Only needed to decide whether the mode switch shows at all, so it's
	// skipped above when there's nothing to show regardless.
	promotions, err := uc.belts.ListBeltPromotions(ctx, userID)
	if err != nil {
		return fmt.Errorf("list belt promotions: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	now := time.Now()
	c := countStats(filterByPeriod(all, p, now))
	weeks := streak(all, now)
	showModeSwitch := len(promotions) >= 2

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, statsText(p, now, c, weeks), statsKeyboard(p, showModeSwitch),
	)
}

func (uc *UseCase) handleBelts(ctx context.Context, userID int64, in dto.Input) error {
	promotions, err := uc.belts.ListBeltPromotions(ctx, userID)
	if err != nil {
		return fmt.Errorf("list belt promotions: %w", err)
	}

	// Nothing to switch between — route back to the default period view
	// rather than showing a belts screen that can only ever be 100% one
	// belt.
	if len(promotions) < 2 {
		return uc.handlePeriod(ctx, userID, in, periodWeek)
	}

	all, err := uc.repo.ListAllTrainings(ctx, userID)
	if err != nil {
		return fmt.Errorf("list all trainings: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	if len(all) == 0 {
		return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, emptyStateText(), emptyStateKeyboard())
	}

	c := countByBelt(all, promotions)
	weeks := streak(all, time.Now())

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, beltsText(c, promotions, weeks), beltsKeyboard())
}
