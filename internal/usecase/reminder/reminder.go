// Package reminder owns two ticker-driven nudges (Run, called every 15
// minutes by cmd/service/main.go — no separate binary, no systemd unit)
// sharing one on/off switch and one idempotency log (reminder_log):
//   - weekly: once a week, if the schedule says a user should have
//     trained by now and they haven't.
//   - post_session: after a scheduled slot ends, offering to record it.
//
// Handle covers both the "🔕 Отключить напоминания" button (shared by
// both kinds' own messages) and the post-session message's own "✅
// Записать" button (rec:{slot_id}:{yyyymmdd}).
package reminder

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/usertime"
	traininginfo "github.com/khambek-archakov/jitLog/internal/usecase/training/info"
)

const (
	// reminderLeadTime is how far before the slot the weekly reminder
	// would ideally fire — clamped to never go earlier than 09:00 local.
	reminderLeadTime = 3 * time.Hour

	sendWindowFloor = 9 // 09:00 local — never remind before this, regardless of how early the slot is.

	// postSessionLeadTime is how long after a session ends the
	// post-session nudge would ideally fire.
	postSessionLeadTime = 15 * time.Minute
	// postSessionWindow is how long the post-session send window stays
	// open — past it, a missed nudge is simply dropped, never sent late.
	postSessionWindow = 6 * time.Hour
	// postSessionWindowStart/End bound the local hours a post-session
	// nudge may fire in — outside them, its send time is clamped to the
	// nearest (future) 09:00.
	postSessionWindowStart = 9
	postSessionWindowEnd   = 22

	// maxRemindersPerDay caps weekly and post-session combined, per the
	// user's own local calendar day.
	maxRemindersPerDay = 2

	// inactivityThreshold: a user this quiet gets neither kind of
	// reminder — nudging someone who's already gone dark is pointless,
	// and winback (internal/usecase/winback) owns that case instead.
	inactivityThreshold = 14 * 24 * time.Hour
)

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

// Run walks every user with reminders_enabled, checking both reminder
// kinds for each. One user's (or one slot's) failure is logged and
// skipped rather than aborting the rest of the walk.
func (uc *UseCase) Run(ctx context.Context) error {
	users, err := uc.users.ListRemindersEnabled(ctx)
	if err != nil {
		return fmt.Errorf("list reminders enabled users: %w", err)
	}

	for _, u := range users {
		uc.processUser(ctx, u)
	}

	return nil
}

func (uc *UseCase) processUser(ctx context.Context, u *model.User) {
	// Precondition from the spec: timezone must actually be set — unlike
	// usertime.Now's general Moscow fallback (used for dates/countdowns
	// elsewhere), a reminder job has no business guessing a zone for
	// someone who never picked one.
	if u.Timezone == nil {
		return
	}

	if uc.now().Sub(u.LastActivityAt) > inactivityThreshold {
		return
	}

	slots, err := uc.slots.ListSlots(ctx, u.ID)
	if err != nil {
		uc.logger.Error("reminder: failed to list schedule slots", "user_id", u.ID, "error", err)

		return
	}

	if len(slots) == 0 {
		return
	}

	if err := uc.processWeekly(ctx, u, slots); err != nil {
		uc.logger.Error("reminder: failed weekly check", "user_id", u.ID, "error", err)
	}

	uc.processPostSession(ctx, u, slots)
}

func (uc *UseCase) processWeekly(ctx context.Context, u *model.User, slots []*model.ScheduleSlot) error {
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

	alreadySent, err := uc.sent.HasSent(ctx, u.ID, model.ReminderKindWeekly, weekStart, 0)
	if err != nil {
		return fmt.Errorf("check reminder sent: %w", err)
	}

	if alreadySent {
		return nil
	}

	withinCap, err := uc.withinDailyCap(ctx, u, now, loc)
	if err != nil {
		return err
	}

	if !withinCap {
		return nil
	}

	messageID, err := uc.bot.SendWithKeyboardReturningID(ctx, u.TelegramID, reminderText(slot), reminderKeyboard())
	if err != nil {
		return uc.handleSendError(ctx, u, err)
	}

	if err := uc.sent.MarkSent(ctx, u.ID, model.ReminderKindWeekly, weekStart, 0, int64(messageID)); err != nil {
		return fmt.Errorf("mark reminder sent: %w", err)
	}

	return nil
}

// processPostSession walks every slot whose weekday matches today or
// yesterday (local) — each one is an independent candidate; one failing
// is logged and doesn't stop the rest from being checked.
func (uc *UseCase) processPostSession(ctx context.Context, u *model.User, slots []*model.ScheduleSlot) {
	loc := usertime.Location(u)
	now := uc.now().In(loc)
	today := startOfDay(now)
	yesterday := today.AddDate(0, 0, -1)

	for _, slot := range slots {
		var d time.Time

		switch {
		case isoWeekday(today) == slot.DayOfWeek:
			d = today
		case isoWeekday(yesterday) == slot.DayOfWeek:
			d = yesterday
		default:
			continue
		}

		if err := uc.processPostSessionSlot(ctx, u, slots, slot, d, now, loc); err != nil {
			uc.logger.Error("reminder: failed post-session check", "user_id", u.ID, "slot_id", slot.ID, "error", err)
		}
	}
}

func (uc *UseCase) processPostSessionSlot(
	ctx context.Context, u *model.User, allSlots []*model.ScheduleSlot, slot *model.ScheduleSlot, d, now time.Time,
	loc *time.Location,
) error {
	n := ordinalOnDay(allSlots, slot)

	slotStart := d.Add(time.Duration(slot.TimeMinutes) * time.Minute)
	end := slotStart.Add(time.Duration(slot.DurationMinutes) * time.Minute)
	sendAt := clampPostSessionSendAt(end.Add(postSessionLeadTime), loc)

	if now.Before(sendAt) || !now.Before(sendAt.Add(postSessionWindow)) {
		return nil // outside [sendAt, sendAt+6h).
	}

	dUTC := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)

	count, err := uc.trainings.CountTrainingsOnDate(ctx, u.ID, dUTC)
	if err != nil {
		return fmt.Errorf("count trainings on date: %w", err)
	}

	if count >= n {
		return nil // this (and every earlier) slot of the day is already covered.
	}

	alreadySent, err := uc.sent.HasSent(ctx, u.ID, model.ReminderKindPostSession, dUTC, slot.ID)
	if err != nil {
		return fmt.Errorf("check reminder sent: %w", err)
	}

	if alreadySent {
		return nil
	}

	withinCap, err := uc.withinDailyCap(ctx, u, now, loc)
	if err != nil {
		return err
	}

	if !withinCap {
		return nil
	}

	// Strip the previous reminder's keyboard (any kind) before sending a
	// new one — best-effort: "message not found" and similar errors are
	// logged, never fatal (a stale/already-gone message must not block
	// this send).
	if lastID, ok, err := uc.sent.LastMessageID(ctx, u.ID); err != nil {
		return fmt.Errorf("get last reminder message id: %w", err)
	} else if ok {
		if err := uc.bot.RemoveKeyboard(ctx, u.TelegramID, int(lastID)); err != nil {
			uc.logger.Warn("reminder: failed to remove previous reminder keyboard", "user_id", u.ID, "error", err)
		}
	}

	isYesterday := d.Before(startOfDay(now))

	messageID, err := uc.bot.SendWithKeyboardReturningID(
		ctx, u.TelegramID, postSessionText(slot, isYesterday), postSessionKeyboard(slot, dUTC),
	)
	if err != nil {
		return uc.handleSendError(ctx, u, err)
	}

	if err := uc.sent.MarkSent(ctx, u.ID, model.ReminderKindPostSession, dUTC, slot.ID, int64(messageID)); err != nil {
		return fmt.Errorf("mark reminder sent: %w", err)
	}

	return nil
}

// withinDailyCap reports whether another reminder (either kind) can still
// be sent today without exceeding maxRemindersPerDay.
func (uc *UseCase) withinDailyCap(ctx context.Context, u *model.User, now time.Time, loc *time.Location) (bool, error) {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.AddDate(0, 0, 1)

	sentToday, err := uc.sent.CountSentBetween(ctx, u.ID, dayStart, dayEnd)
	if err != nil {
		return false, fmt.Errorf("count reminders sent today: %w", err)
	}

	return sentToday < maxRemindersPerDay, nil
}

// handleSendError classifies a send failure: a blocked bot disables
// reminders (logged, never retried); anything else is just logged, also
// never retried within this same run.
func (uc *UseCase) handleSendError(ctx context.Context, u *model.User, err error) error {
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

// Handle covers the "🔕 Отключить напоминания" button (shared by both
// reminder kinds) and the post-session message's own "✅ Записать" button.
func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	switch {
	case in.CallbackData == callbackDisable:
		return uc.handleDisable(ctx, u, in)

	case strings.HasPrefix(in.CallbackData, callbackRecPrefix):
		return uc.handleRec(ctx, u, in)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

func (uc *UseCase) handleDisable(ctx context.Context, u *model.User, in dto.Input) error {
	u.RemindersEnabled = false

	if err := uc.users.Update(ctx, u); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, disabledText, nil)
}

// handleRec covers rec:{slot_id}:{yyyymmdd} — "✅ Записать" on a
// post-session message. Tapping it is itself activity, same as any other
// callback — Router's own activity middleware already covers that, no
// extra code needed here.
func (uc *UseCase) handleRec(ctx context.Context, u *model.User, in dto.Input) error {
	slot, date, ok, err := uc.parseRecCallback(ctx, in.CallbackData, u.ID)
	if err != nil {
		return fmt.Errorf("get schedule slot: %w", err)
	}

	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	// Deliberately not usertime.Today(u) — that always reads the real wall
	// clock, bypassing uc.now's own test seam.
	nowLocal := uc.now().In(usertime.Location(u))
	today := time.Date(nowLocal.Year(), nowLocal.Month(), nowLocal.Day(), 0, 0, 0, 0, time.UTC)

	if date.After(today) || date.Before(today.AddDate(0, 0, -3)) {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, "Эту тренировку уже не записать отсюда.")
	}

	allSlots, err := uc.slots.ListSlots(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("list schedule slots: %w", err)
	}

	n := ordinalOnDay(allSlots, slot)

	count, err := uc.trainings.CountTrainingsOnDate(ctx, u.ID, date)
	if err != nil {
		return fmt.Errorf("count trainings on date: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	if count >= n {
		return uc.showExistingTraining(ctx, u, in, date)
	}

	t, err := uc.trainings.CreateTraining(ctx, u.ID, date, slot.TrainingType, int32(slot.DurationMinutes), nil)
	if err != nil {
		return fmt.Errorf("create training: %w", err)
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, traininginfo.ConfirmationText(t), traininginfo.ConfirmationKeyboard(t.ID),
	)
}

// parseRecCallback validates the callback down to the spec's own list:
// well-formed, the slot exists, and it belongs to this user. The final
// bool is false for every "reject silently" case (malformed data, no
// such slot, someone else's slot) — err is only ever set for a genuine
// repository failure, which the caller treats as a real error.
func (uc *UseCase) parseRecCallback(
	ctx context.Context, data string, userID int64,
) (*model.ScheduleSlot, time.Time, bool, error) {
	rest := strings.TrimPrefix(data, callbackRecPrefix)

	idStr, dateStr, found := strings.Cut(rest, ":")
	if !found {
		return nil, time.Time{}, false, nil
	}

	slotID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return nil, time.Time{}, false, nil
	}

	date, ok := parseRecDate(dateStr)
	if !ok {
		return nil, time.Time{}, false, nil
	}

	slot, err := uc.slots.GetSlot(ctx, slotID)
	if errors.Is(err, model.ErrNotFound) {
		return nil, time.Time{}, false, nil
	}
	if err != nil {
		return nil, time.Time{}, false, err
	}

	if slot.UserID != userID {
		return nil, time.Time{}, false, nil
	}

	return slot, date, true, nil
}

func (uc *UseCase) showExistingTraining(ctx context.Context, u *model.User, in dto.Input, date time.Time) error {
	existing, err := uc.trainings.LatestTrainingOnDate(ctx, u.ID, date)
	if err != nil {
		return fmt.Errorf("get latest training on date: %w", err)
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, traininginfo.Card(existing), traininginfo.Keyboard(existing.ID),
	)
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

// ordinalOnDay is target's 1-based position among allSlots sharing its
// own DayOfWeek, ordered by TimeMinutes — slotRepo.ListSlots already
// returns slots in that order, so no re-sorting is needed here.
func ordinalOnDay(allSlots []*model.ScheduleSlot, target *model.ScheduleSlot) int {
	n := 0

	for _, s := range allSlots {
		if s.DayOfWeek != target.DayOfWeek {
			continue
		}

		n++

		if s.ID == target.ID {
			return n
		}
	}

	return n
}

// clampPostSessionSendAt pushes instant forward to the nearest future
// 09:00 local when it would otherwise fall before 09:00 or at/after
// 22:00 — a post-session nudge never fires at night.
func clampPostSessionSendAt(instant time.Time, loc *time.Location) time.Time {
	local := instant.In(loc)

	if local.Hour() >= postSessionWindowEnd {
		next := local.AddDate(0, 0, 1)

		return time.Date(next.Year(), next.Month(), next.Day(), postSessionWindowStart, 0, 0, 0, loc)
	}

	if local.Hour() < postSessionWindowStart {
		return time.Date(local.Year(), local.Month(), local.Day(), postSessionWindowStart, 0, 0, 0, loc)
	}

	return instant
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
