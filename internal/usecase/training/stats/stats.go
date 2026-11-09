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
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/usertime"
)

type UseCase struct {
	bot   sender
	repo  trainingRepo
	belts beltRepo
	now   func() time.Time
}

func New(bot sender, repo trainingRepo, belts beltRepo) *UseCase {
	return NewWithClock(bot, repo, belts, time.Now)
}

// NewWithClock lets tests substitute a fixed instant instead of the real
// wall clock — the calendar's own "today"/"first available month" bounds
// need it to be deterministic across timezones. See
// usecase/competition/import/ajp's identical pattern.
func NewWithClock(bot sender, repo trainingRepo, belts beltRepo, now func() time.Time) *UseCase {
	return &UseCase{bot: bot, repo: repo, belts: belts, now: now}
}

func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	switch {
	case in.CallbackData == callbackStatsBelts:
		return uc.handleBelts(ctx, u, in)

	case strings.HasPrefix(in.CallbackData, callbackSharePrefix):
		return uc.handleShare(ctx, u, in)

	case strings.HasPrefix(in.CallbackData, callbackCalPrefix):
		return uc.handleCalendar(ctx, u, in)
	}

	p, ok := parsePeriod(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	return uc.handlePeriod(ctx, u, in, p)
}

func (uc *UseCase) handlePeriod(ctx context.Context, u *model.User, in dto.Input, p period) error {
	all, err := uc.repo.ListAllTrainings(ctx, u.ID)
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
	promotions, err := uc.belts.ListBeltPromotions(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("list belt promotions: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	now := uc.now().In(usertime.Location(u))
	c := countStats(filterByPeriod(all, p, now))
	weeks := streak(all, now)
	showModeSwitch := len(promotions) >= 2
	showShare := (p == periodMonth || p == periodYear) && c.Total > 0

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, statsText(p, now, c, weeks), statsKeyboard(p, showModeSwitch, showShare, now),
	)
}

func (uc *UseCase) handleBelts(ctx context.Context, u *model.User, in dto.Input) error {
	promotions, err := uc.belts.ListBeltPromotions(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("list belt promotions: %w", err)
	}

	// Nothing to switch between — route back to the default period view
	// rather than showing a belts screen that can only ever be 100% one
	// belt.
	if len(promotions) < 2 {
		return uc.handlePeriod(ctx, u, in, periodWeek)
	}

	all, err := uc.repo.ListAllTrainings(ctx, u.ID)
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
	weeks := streak(all, uc.now().In(usertime.Location(u)))

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, beltsText(c, promotions, weeks), beltsKeyboard())
}

// handleCalendar covers every cal:* callback — the "📅 Другие даты"
// custom-range picker (see calendar.go). Nothing here is stored: each tap
// re-derives everything from the callback data itself, so an old message
// (even from before a restart) keeps working, and a stale/repeat tap is
// always safe to re-render. Every date within [calAbsoluteMin,
// calAbsoluteMax] is pickable — there's no "unavailable day" case left to
// dispatch on.
func (uc *UseCase) handleCalendar(ctx context.Context, u *model.User, in dto.Input) error {
	switch {
	case in.CallbackData == callbackCalNoop:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)

	case in.CallbackData == callbackCalCancel:
		return uc.handlePeriod(ctx, u, in, periodWeek)

	case strings.HasPrefix(in.CallbackData, callbackCalFromMonthPrefix):
		return uc.handleCalFromMonth(ctx, in)

	case strings.HasPrefix(in.CallbackData, callbackCalFromDayPrefix):
		return uc.handleCalFromDay(ctx, in)

	case strings.HasPrefix(in.CallbackData, callbackCalToMonthPrefix):
		return uc.handleCalToMonth(ctx, in)

	case strings.HasPrefix(in.CallbackData, callbackCalToDayPrefix):
		return uc.handleCalToDay(ctx, in)

	case strings.HasPrefix(in.CallbackData, callbackCalApplyPrefix):
		return uc.handleCalApply(ctx, u, in)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

func (uc *UseCase) handleCalFromMonth(ctx context.Context, in dto.Input) error {
	month, ok := parseCalFromMonth(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboardMarkdown(
		ctx, in.ChatID, in.MessageID, calFromText(month), calFromKeyboard(month),
	)
}

func (uc *UseCase) handleCalFromDay(ctx context.Context, in dto.Input) error {
	day, ok := parseCalFromDay(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboardMarkdown(
		ctx, in.ChatID, in.MessageID, calToText(day, day), calToKeyboard(day, day),
	)
}

func (uc *UseCase) handleCalToMonth(ctx context.Context, in dto.Input) error {
	from, month, ok := parseCalToMonth(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboardMarkdown(
		ctx, in.ChatID, in.MessageID, calToText(from, month), calToKeyboard(from, month),
	)
}

// handleCalToDay picks the "До" day, landing on the confirm screen. A
// "До" earlier than "От" is no longer rejected — it's swapped, per the
// spec, so the pair always continues as if picked in the right order.
func (uc *UseCase) handleCalToDay(ctx context.Context, in dto.Input) error {
	from, to, ok := parseCalToDay(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	if to.Before(from) {
		from, to = to, from
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, calConfirmText(from, to), calConfirmKeyboard(from, to),
	)
}

func (uc *UseCase) handleCalApply(ctx context.Context, u *model.User, in dto.Input) error {
	from, to, ok := parseCalApply(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	all, err := uc.repo.ListAllTrainings(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("list all trainings: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	c := countStats(filterByRange(all, from, to))

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, rangeText(from, to, c), rangeKeyboard(uc.now().In(usertime.Location(u))),
	)
}
