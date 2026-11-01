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

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, statsText(p, now, c, weeks), statsKeyboard(p, showModeSwitch, now),
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
// re-derives everything from the callback data itself plus a fresh
// calendarBounds call, so an old message (even from before a restart)
// keeps working, and a stale/repeat tap is always safe to re-render.
func (uc *UseCase) handleCalendar(ctx context.Context, u *model.User, in dto.Input) error {
	switch {
	case in.CallbackData == callbackCalNoop:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)

	case in.CallbackData == callbackCalUnavailable:
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, calUnavailableToast)

	case in.CallbackData == callbackCalCancel:
		return uc.handlePeriod(ctx, u, in, periodWeek)

	case strings.HasPrefix(in.CallbackData, callbackCalFromMonthPrefix):
		return uc.handleCalFromMonth(ctx, u, in)

	case strings.HasPrefix(in.CallbackData, callbackCalFromDayPrefix):
		return uc.handleCalFromDay(ctx, u, in)

	case strings.HasPrefix(in.CallbackData, callbackCalToMonthPrefix):
		return uc.handleCalToMonth(ctx, u, in)

	case strings.HasPrefix(in.CallbackData, callbackCalToDayPrefix):
		return uc.handleCalToDay(ctx, u, in)

	case strings.HasPrefix(in.CallbackData, callbackCalApplyPrefix):
		return uc.handleCalApply(ctx, u, in)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

func (uc *UseCase) handleCalFromMonth(ctx context.Context, u *model.User, in dto.Input) error {
	month, ok := parseCalFromMonth(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	minDate, maxDate, err := uc.calendarBounds(ctx, u)
	if err != nil {
		return err
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, calFromText(), calFromKeyboard(month, minDate, maxDate),
	)
}

func (uc *UseCase) handleCalFromDay(ctx context.Context, u *model.User, in dto.Input) error {
	day, ok := parseCalFromDay(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	minDate, maxDate, err := uc.calendarBounds(ctx, u)
	if err != nil {
		return err
	}

	if day.Before(minDate) || day.After(maxDate) {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, calUnavailableToast)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, calToText(day), calToKeyboard(day, day, minDate, maxDate),
	)
}

func (uc *UseCase) handleCalToMonth(ctx context.Context, u *model.User, in dto.Input) error {
	from, month, ok := parseCalToMonth(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	minDate, maxDate, err := uc.calendarBounds(ctx, u)
	if err != nil {
		return err
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, calToText(from), calToKeyboard(from, month, minDate, maxDate),
	)
}

func (uc *UseCase) handleCalToDay(ctx context.Context, u *model.User, in dto.Input) error {
	from, to, ok := parseCalToDay(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	minDate, maxDate, err := uc.calendarBounds(ctx, u)
	if err != nil {
		return err
	}

	if to.Before(from) || to.Before(minDate) || to.After(maxDate) {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, calUnavailableToast)
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

// calendarBounds is the calendar's overall selectable range: never before
// the user's first training (or, with none yet, their own signup date —
// see minAvailableDate), never after today in the user's own timezone.
func (uc *UseCase) calendarBounds(ctx context.Context, u *model.User) (time.Time, time.Time, error) {
	all, err := uc.repo.ListAllTrainings(ctx, u.ID)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("list all trainings: %w", err)
	}

	now := uc.now().In(usertime.Location(u))

	return minAvailableDate(u, all), today(now), nil
}

// minAvailableDate walks all rather than trusting it's sorted oldest-first
// (ListAllTrainings' own doc comment promises that, but this doesn't need
// to depend on it to stay correct).
func minAvailableDate(u *model.User, all []*model.Training) time.Time {
	createdAt := u.CreatedAt.UTC()
	floor := time.Date(createdAt.Year(), createdAt.Month(), createdAt.Day(), 0, 0, 0, 0, time.UTC)

	for _, t := range all {
		if t.Date.Before(floor) {
			floor = t.Date
		}
	}

	return floor
}
