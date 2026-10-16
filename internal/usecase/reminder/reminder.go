// Package reminder owns the weekly training reminder: a ticker-driven job
// (Run, called every 15 minutes by cmd/service/main.go — no separate
// binary, no systemd unit) that nudges a user once a week if their
// schedule says they should have trained by now and they haven't, plus the
// "🔕 Отключить напоминания" button's own callback (Handle) on whatever
// message it sends.
package reminder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/usertime"
)

// reminderLeadTime is how far before the slot the reminder would ideally
// fire — clamped to never go earlier than 09:00 local (see processUser).
const reminderLeadTime = 3 * time.Hour

const sendWindowFloor = 9 // 09:00 local — never remind before this, regardless of how early the slot is.

type UseCase struct {
	bot       sender
	users     userRepo
	slots     slotRepo
	trainings trainingRepo
	sent      sentRepo
	logger    *slog.Logger
	now       func() time.Time
}

func New(bot sender, users userRepo, slots slotRepo, trainings trainingRepo, sent sentRepo, logger *slog.Logger) *UseCase {
	return NewWithClock(bot, users, slots, trainings, sent, logger, time.Now)
}

// NewWithClock lets tests substitute a fixed instant instead of the real
// wall clock — see usecase/competition/import/ajp's identical pattern.
func NewWithClock(
	bot sender, users userRepo, slots slotRepo, trainings trainingRepo, sent sentRepo, logger *slog.Logger,
	now func() time.Time,
) *UseCase {
	return &UseCase{bot: bot, users: users, slots: slots, trainings: trainings, sent: sent, logger: logger, now: now}
}

// Run walks every user with reminders_enabled and sends at most one
// reminder per user. One user's failure is logged and skipped rather than
// aborting the rest of the walk.
func (uc *UseCase) Run(ctx context.Context) error {
	users, err := uc.users.ListRemindersEnabled(ctx)
	if err != nil {
		return fmt.Errorf("list reminders enabled users: %w", err)
	}

	for _, u := range users {
		if err := uc.processUser(ctx, u); err != nil {
			uc.logger.Error("reminder: failed to process user", "user_id", u.ID, "error", err)
		}
	}

	return nil
}

func (uc *UseCase) processUser(ctx context.Context, u *model.User) error {
	// Precondition from the spec: timezone must actually be set — unlike
	// usertime.Now's general Moscow fallback (used for dates/countdowns
	// elsewhere), a reminder job has no business guessing a zone for
	// someone who never picked one.
	if u.Timezone == nil {
		return nil
	}

	slots, err := uc.slots.ListSlots(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("list schedule slots: %w", err)
	}

	if len(slots) == 0 {
		return nil
	}

	loc := usertime.Location(u)
	now := uc.now().In(loc)

	lastDay := lastScheduledDay(slots)
	if isoWeekday(now) != lastDay {
		return nil
	}

	slot := earliestSlotOnDay(slots, lastDay)

	slotTime := startOfDay(now).Add(time.Duration(slot.TimeMinutes) * time.Minute)
	target := slotTime.Add(-reminderLeadTime)

	floor := time.Date(now.Year(), now.Month(), now.Day(), sendWindowFloor, 0, 0, 0, loc)
	if target.Before(floor) {
		target = floor
	}

	if !target.Before(slotTime) {
		return nil // clamped target caught up to (or passed) the slot — reminder would be meaningless.
	}

	if now.Before(target) || !now.Before(slotTime) {
		return nil // outside [target, slot).
	}

	weekStart := usertime.StartOfWeek(now)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	trained, err := uc.trainings.HasTrainingBetween(ctx, u.ID, weekStart, today)
	if err != nil {
		return fmt.Errorf("check training this week: %w", err)
	}

	if trained {
		return nil
	}

	alreadySent, err := uc.sent.HasSent(ctx, u.ID, weekStart)
	if err != nil {
		return fmt.Errorf("check reminder sent: %w", err)
	}

	if alreadySent {
		return nil
	}

	if err := uc.bot.SendWithKeyboard(ctx, u.TelegramID, reminderText(slot), reminderKeyboard()); err != nil {
		if errors.Is(err, model.ErrUserBlockedBot) {
			u.RemindersEnabled = false

			if upErr := uc.users.Update(ctx, u); upErr != nil {
				return fmt.Errorf("disable reminders for blocked user: %w", upErr)
			}

			uc.logger.Info("reminder: user blocked the bot, reminders disabled", "user_id", u.ID)

			return nil
		}

		return fmt.Errorf("send reminder: %w", err)
	}

	if err := uc.sent.MarkSent(ctx, u.ID, weekStart); err != nil {
		return fmt.Errorf("mark reminder sent: %w", err)
	}

	return nil
}

// Handle covers the "🔕 Отключить напоминания" button on whatever
// reminder message Run sent.
func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	if in.CallbackData != callbackDisable {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	u.RemindersEnabled = false

	if err := uc.users.Update(ctx, u); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, disabledText, nil)
}

func lastScheduledDay(slots []*model.ScheduleSlot) int16 {
	var last int16

	for _, s := range slots {
		if s.DayOfWeek > last {
			last = s.DayOfWeek
		}
	}

	return last
}

func earliestSlotOnDay(slots []*model.ScheduleSlot, day int16) *model.ScheduleSlot {
	var earliest *model.ScheduleSlot

	for _, s := range slots {
		if s.DayOfWeek != day {
			continue
		}

		if earliest == nil || s.TimeMinutes < earliest.TimeMinutes {
			earliest = s
		}
	}

	return earliest
}

// isoWeekday converts Go's own 0=Sunday..6=Saturday numbering to
// schedule_slot's 1=Monday..7=Sunday.
func isoWeekday(t time.Time) int16 {
	wd := int16(t.Weekday())
	if wd == 0 {
		wd = 7
	}

	return wd
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
