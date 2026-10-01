// Package stats owns the "📊 Статистика" screen — read-only, no drafts,
// one entry point for both the main menu's own button (which already
// points at stats:period:week) and every period-tab button on the screen
// itself.
package stats

import (
	"context"
	"fmt"
	"time"

	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

type UseCase struct {
	bot  sender
	repo trainingRepo
}

func New(bot sender, repo trainingRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

func (uc *UseCase) Handle(ctx context.Context, userID int64, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	p, ok := parsePeriod(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(in.CallbackID)
	}

	all, err := uc.repo.ListAllTrainings(ctx, userID)
	if err != nil {
		return fmt.Errorf("list all trainings: %w", err)
	}

	if err := uc.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	if len(all) == 0 {
		return uc.bot.EditMessageWithKeyboard(in.ChatID, in.MessageID, emptyStateText(), emptyStateKeyboard())
	}

	now := time.Now()
	c := countStats(filterByPeriod(all, p, now))
	weeks := streak(all, now)

	return uc.bot.EditMessageWithKeyboard(in.ChatID, in.MessageID, statsText(p, now, c, weeks), statsKeyboard(p))
}
