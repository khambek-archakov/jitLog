package reminder_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/reminder"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func ptr[T any](v T) *T {
	return &v
}

const (
	userID, telegramID int64 = 1, 999
	chatID, messageID        = telegramID, 777
)

// friWeekStart/friToday mirror the reminder feature's own earlier fixture
// — 2026-10-16 is a Friday, 2026-10-12 its Monday.
var (
	friWeekStart = time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	friToday     = time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
)

type mocks struct {
	bot       *Mocksender
	users     *MockuserRepo
	slots     *MockslotRepo
	trainings *MocktrainingRepo
	sent      *MocksentRepo
}

func newMocks(ctrl *gomock.Controller) mocks {
	return mocks{
		bot:       NewMocksender(ctrl),
		users:     NewMockuserRepo(ctrl),
		slots:     NewMockslotRepo(ctrl),
		trainings: NewMocktrainingRepo(ctrl),
		sent:      NewMocksentRepo(ctrl),
	}
}

func newUseCase(m mocks, clock time.Time) *reminder.UseCase {
	return reminder.NewWithClock(m.bot, m.users, m.slots, m.trainings, m.sent, discardLogger(), func() time.Time { return clock })
}

func moscowUser(lastActivityAt time.Time) *model.User {
	return &model.User{
		ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Moscow"), RemindersEnabled: true,
		LastActivityAt: lastActivityAt,
	}
}

// noDailyCapHit stubs the shared "fewer than 2 reminders sent today" check
// to always allow — used by every test that isn't specifically about the
// cap itself.
func noDailyCapHit(m mocks) {
	m.sent.EXPECT().CountSentBetween(gomock.Any(), userID, gomock.Any(), gomock.Any()).Return(0, nil).AnyTimes()
}

func TestRun_Weekly(t *testing.T) {
	t.Parallel()

	// Friday 17:00 Moscow (+3) == 14:00 UTC — inside [16:00, 19:00).
	now := time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)

	friSlot := func() []*model.ScheduleSlot {
		return []*model.ScheduleSlot{{ID: 50, DayOfWeek: 5, TimeMinutes: 19 * 60, DurationMinutes: 60, TrainingType: model.TrainingTypeGi}}
	}

	t.Run("sends inside the window, no prior training, not yet sent", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindWeekly, friWeekStart, int64(0)).Return(false, nil)
		noDailyCapHit(m)

		m.bot.EXPECT().
			SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, text string, kb dto.Keyboard) (int, error) {
				assert.Contains(t, text, "19:00")
				require.Len(t, kb, 1)
				assert.Equal(t, "reminder:disable", kb[0][0].Data)

				return 555, nil
			})

		m.sent.EXPECT().MarkSent(gomock.Any(), userID, model.ReminderKindWeekly, friWeekStart, int64(0), int64(555)).Return(nil)

		// post-session: no slot matches Friday/Thursday... wait Friday
		// IS today, so the post-session walk also considers this same
		// slot — stub it as "not due" by making its own window check
		// fail naturally: 19:00+60min+15min lead = 20:15, well after
		// 14:00 "now", so processPostSessionSlot returns before any
		// further mock calls.

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("already trained this week — no reminder", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(true, nil)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("already sent this week — re-running in the same window sends no duplicate", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindWeekly, friWeekStart, int64(0)).Return(true, nil)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a slot only on Monday fires on Monday", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		mondayNow := time.Date(2026, 10, 19, 14, 0, 0, 0, time.UTC) // Monday 17:00 Moscow
		mondayWeekStart := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)

		u := moscowUser(mondayNow.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().
			ListSlots(gomock.Any(), userID).
			Return([]*model.ScheduleSlot{{ID: 51, DayOfWeek: 1, TimeMinutes: 19 * 60, DurationMinutes: 60, TrainingType: model.TrainingTypeNoGi}}, nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, mondayWeekStart, mondayWeekStart).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindWeekly, mondayWeekStart, int64(0)).Return(false, nil)
		noDailyCapHit(m)
		m.bot.EXPECT().SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).Return(555, nil)
		m.sent.EXPECT().MarkSent(gomock.Any(), userID, model.ReminderKindWeekly, mondayWeekStart, int64(0), int64(555)).Return(nil)

		err := newUseCase(m, mondayNow).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("no schedule at all — no reminder of either kind", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(nil, nil)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("last_activity_at older than 14 days — neither kind, schedule never even listed", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-15 * 24 * time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		// No ListSlots call at all.

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("the daily cap (weekly and post-session combined) blocks a second reminder", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindWeekly, friWeekStart, int64(0)).Return(false, nil)
		m.sent.EXPECT().CountSentBetween(gomock.Any(), userID, gomock.Any(), gomock.Any()).Return(2, nil)
		// No Send/MarkSent at all.

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a blocked user has reminders disabled and is not retried this run", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindWeekly, friWeekStart, int64(0)).Return(false, nil)
		noDailyCapHit(m)
		m.bot.EXPECT().
			SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).
			Return(0, model.ErrUserBlockedBot)
		m.users.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, u *model.User) error {
				assert.False(t, u.RemindersEnabled)

				return nil
			})

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("local time window: Europe/Kaliningrad", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		// 13:00 UTC == 15:00 Kaliningrad (+2) — inside [16:00, 19:00)?
		// No — pick a Kaliningrad-local-correct instant instead: slot at
		// 19:00 Kaliningrad local, window [16:00, 19:00) Kaliningrad.
		// 16:30 Kaliningrad == 14:30 UTC.
		clock := time.Date(2026, 10, 16, 14, 30, 0, 0, time.UTC)

		u := &model.User{
			ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Kaliningrad"), RemindersEnabled: true,
			LastActivityAt: clock.Add(-time.Hour),
		}
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindWeekly, friWeekStart, int64(0)).Return(false, nil)
		noDailyCapHit(m)
		m.bot.EXPECT().SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).Return(555, nil)
		m.sent.EXPECT().MarkSent(gomock.Any(), userID, model.ReminderKindWeekly, friWeekStart, int64(0), int64(555)).Return(nil)

		err := newUseCase(m, clock).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("local time window: Asia/Yekaterinburg", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		// 17:00 Yekaterinburg (+5) == 12:00 UTC — inside [16:00, 19:00).
		clock := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

		u := &model.User{
			ID: userID, TelegramID: telegramID, Timezone: ptr("Asia/Yekaterinburg"), RemindersEnabled: true,
			LastActivityAt: clock.Add(-time.Hour),
		}
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindWeekly, friWeekStart, int64(0)).Return(false, nil)
		noDailyCapHit(m)
		m.bot.EXPECT().SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).Return(555, nil)
		m.sent.EXPECT().MarkSent(gomock.Any(), userID, model.ReminderKindWeekly, friWeekStart, int64(0), int64(555)).Return(nil)

		err := newUseCase(m, clock).Run(context.Background())
		assert.NoError(t, err)
	})
}

func TestRun_PostSession(t *testing.T) {
	t.Parallel()

	oneSlot := func() []*model.ScheduleSlot {
		return []*model.ScheduleSlot{{ID: 60, DayOfWeek: 5, TimeMinutes: 19 * 60, DurationMinutes: 60, TrainingType: model.TrainingTypeGi}}
	}

	// Friday 20:20 Moscow (+3) == 17:20 UTC — 15 minutes after the slot
	// (19:00-20:00) ends, well inside [20:15, 02:15 next day).
	sendWindowNow := time.Date(2026, 10, 16, 17, 20, 0, 0, time.UTC)

	t.Run("one slot, no record yet — sends 15 minutes after it ends", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(sendWindowNow.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
		// Weekly's own window (same slot) is still open too (target 16:00
		// clamp-free, slot 19:00) — but "now" (20:20) is past the slot,
		// so weekly's own check exits before touching the mocks.
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(0, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60)).Return(false, nil)
		noDailyCapHit(m)
		m.sent.EXPECT().LastMessageID(gomock.Any(), userID).Return(int64(0), false, nil)

		m.bot.EXPECT().
			SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, text string, kb dto.Keyboard) (int, error) {
				assert.Contains(t, text, "Записать тренировку?")
				assert.NotContains(t, text, "Вчера")
				require.Len(t, kb, 2)
				assert.Equal(t, "rec:60:20261016", kb[0][0].Data)
				assert.Equal(t, "✅ Записать (60 мин)", kb[0][0].Label)
				assert.Equal(t, "reminder:disable", kb[1][0].Data)

				return 777, nil
			})
		m.sent.EXPECT().
			MarkSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60), int64(777)).
			Return(nil)

		err := newUseCase(m, sendWindowNow).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a training already logged for that date — no nudge", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(sendWindowNow.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(1, nil)
		// count(1) >= n(1) — no further calls.

		err := newUseCase(m, sendWindowNow).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("two slots in a day: an evening slot still fires with one morning training logged (n=2, count=1)", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		slots := []*model.ScheduleSlot{
			{ID: 61, DayOfWeek: 5, TimeMinutes: 8 * 60, DurationMinutes: 60, TrainingType: model.TrainingTypeGi},  // morning, n=1
			{ID: 62, DayOfWeek: 5, TimeMinutes: 19 * 60, DurationMinutes: 60, TrainingType: model.TrainingTypeGi}, // evening, n=2
		}

		u := moscowUser(sendWindowNow.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(slots, nil)
		// Morning slot's own window (9:15-15:15) is long closed by 20:20
		// — it exits before any mock call. Only the evening slot reaches
		// the count/n check.
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(1, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(62)).Return(false, nil)
		noDailyCapHit(m)
		m.sent.EXPECT().LastMessageID(gomock.Any(), userID).Return(int64(0), false, nil)
		m.bot.EXPECT().SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).Return(777, nil)
		m.sent.EXPECT().MarkSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(62), int64(777)).Return(nil)

		err := newUseCase(m, sendWindowNow).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("two slots in a day: two trainings already logged suppress the evening slot too", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		slots := []*model.ScheduleSlot{
			{ID: 61, DayOfWeek: 5, TimeMinutes: 8 * 60, DurationMinutes: 60, TrainingType: model.TrainingTypeGi},
			{ID: 62, DayOfWeek: 5, TimeMinutes: 19 * 60, DurationMinutes: 60, TrainingType: model.TrainingTypeGi},
		}

		u := moscowUser(sendWindowNow.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(slots, nil)
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(2, nil)
		// count(2) >= n(2) — no further calls.

		err := newUseCase(m, sendWindowNow).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("send_at clamped past 22:00 rolls to the next day's 09:00, text says Вчера", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		// Slot on Wednesday (2026-10-14) at 21:00 for 90 minutes — ends
		// 22:30, +15m lead = 22:45, past 22:00, clamped to Thursday 09:00.
		slot := []*model.ScheduleSlot{{ID: 70, DayOfWeek: 3, TimeMinutes: 21 * 60, DurationMinutes: 90, TrainingType: model.TrainingTypeNoGi}}
		wedDate := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)

		// Thursday 10:00 Moscow (+3) == 07:00 UTC — inside the clamped
		// window [Thu 09:00, Thu 15:00).
		now := time.Date(2026, 10, 15, 7, 0, 0, 0, time.UTC)

		u := moscowUser(now.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(slot, nil)
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, wedDate).Return(0, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindPostSession, wedDate, int64(70)).Return(false, nil)
		noDailyCapHit(m)
		m.sent.EXPECT().LastMessageID(gomock.Any(), userID).Return(int64(0), false, nil)

		m.bot.EXPECT().
			SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, text string, kb dto.Keyboard) (int, error) {
				assert.Contains(t, text, "Вчера, ")
				assert.Equal(t, "rec:70:20261014", kb[0][0].Data)

				return 777, nil
			})
		m.sent.EXPECT().
			MarkSent(gomock.Any(), userID, model.ReminderKindPostSession, wedDate, int64(70), int64(777)).
			Return(nil)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("the bot was unreachable for more than 6 hours — the window closed, nothing sends", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		// 7 hours after the send window opened (20:15 + 7h == 03:15 next day).
		lateNow := sendWindowNow.Add(7 * time.Hour)

		u := moscowUser(lateNow.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
		// No further calls — the [sendAt, sendAt+6h) window has closed.

		err := newUseCase(m, lateNow).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a second run in the same window sends no duplicate", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(sendWindowNow.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(0, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60)).Return(true, nil)
		// Already sent — no further calls.

		err := newUseCase(m, sendWindowNow).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a new reminder strips the previous one's keyboard first", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(sendWindowNow.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(0, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60)).Return(false, nil)
		noDailyCapHit(m)
		m.sent.EXPECT().LastMessageID(gomock.Any(), userID).Return(int64(321), true, nil)
		m.bot.EXPECT().RemoveKeyboard(gomock.Any(), telegramID, 321).Return(nil)
		m.bot.EXPECT().SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).Return(777, nil)
		m.sent.EXPECT().MarkSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60), int64(777)).Return(nil)

		err := newUseCase(m, sendWindowNow).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a failed keyboard removal (e.g. message not found) is logged, never blocks the send", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(sendWindowNow.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(0, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60)).Return(false, nil)
		noDailyCapHit(m)
		m.sent.EXPECT().LastMessageID(gomock.Any(), userID).Return(int64(321), true, nil)
		m.bot.EXPECT().RemoveKeyboard(gomock.Any(), telegramID, 321).Return(errors.New("message to edit not found"))
		m.bot.EXPECT().SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).Return(777, nil)
		m.sent.EXPECT().MarkSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60), int64(777)).Return(nil)

		err := newUseCase(m, sendWindowNow).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a blocked user has reminders disabled and is not retried this run", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(sendWindowNow.Add(-time.Hour))
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(0, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60)).Return(false, nil)
		noDailyCapHit(m)
		m.sent.EXPECT().LastMessageID(gomock.Any(), userID).Return(int64(0), false, nil)
		m.bot.EXPECT().
			SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).
			Return(0, model.ErrUserBlockedBot)
		m.users.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, u *model.User) error {
				assert.False(t, u.RemindersEnabled)

				return nil
			})

		err := newUseCase(m, sendWindowNow).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("local time window: Europe/Kaliningrad", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		// Slot ends 20:00 Kaliningrad local, +15m lead = 20:15. Pick now
		// at 20:20 Kaliningrad (+2) == 18:20 UTC.
		clock := time.Date(2026, 10, 16, 18, 20, 0, 0, time.UTC)

		u := &model.User{
			ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Kaliningrad"), RemindersEnabled: true,
			LastActivityAt: clock.Add(-time.Hour),
		}
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(0, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60)).Return(false, nil)
		noDailyCapHit(m)
		m.sent.EXPECT().LastMessageID(gomock.Any(), userID).Return(int64(0), false, nil)
		m.bot.EXPECT().SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).Return(777, nil)
		m.sent.EXPECT().MarkSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60), int64(777)).Return(nil)

		err := newUseCase(m, clock).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("local time window: Asia/Yekaterinburg", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		// Slot ends 20:00 Yekaterinburg local, +15m lead = 20:15. Pick
		// now at 20:20 Yekaterinburg (+5) == 15:20 UTC.
		clock := time.Date(2026, 10, 16, 15, 20, 0, 0, time.UTC)

		u := &model.User{
			ID: userID, TelegramID: telegramID, Timezone: ptr("Asia/Yekaterinburg"), RemindersEnabled: true,
			LastActivityAt: clock.Add(-time.Hour),
		}
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(oneSlot(), nil)
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(0, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60)).Return(false, nil)
		noDailyCapHit(m)
		m.sent.EXPECT().LastMessageID(gomock.Any(), userID).Return(int64(0), false, nil)
		m.bot.EXPECT().SendWithKeyboardReturningID(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).Return(777, nil)
		m.sent.EXPECT().MarkSent(gomock.Any(), userID, model.ReminderKindPostSession, friToday, int64(60), int64(777)).Return(nil)

		err := newUseCase(m, clock).Run(context.Background())
		assert.NoError(t, err)
	})
}

func TestHandle_Disable(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	m := newMocks(ctrl)

	u := &model.User{ID: userID, TelegramID: telegramID, RemindersEnabled: true}

	m.users.EXPECT().
		Update(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, u *model.User) error {
			assert.False(t, u.RemindersEnabled)

			return nil
		})
	m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
	m.bot.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, messageID, "🔕 Напоминания отключены.", nil).Return(nil)

	in := dto.Input{
		ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "reminder:disable",
	}

	err := newUseCase(m, time.Now()).Handle(context.Background(), u, in)
	assert.NoError(t, err)
}

func TestHandle_Rec(t *testing.T) {
	t.Parallel()

	// "Today" from the handler's own point of view (usertime.Today(u)) —
	// Friday 2026-10-16, Moscow, no particular time of day matters here.
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

	u := &model.User{ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Moscow")}

	slot := &model.ScheduleSlot{ID: 60, UserID: userID, DayOfWeek: 5, TimeMinutes: 19 * 60, DurationMinutes: 60, TrainingType: model.TrainingTypeGi}

	t.Run("records the training on the first tap", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.slots.EXPECT().GetSlot(gomock.Any(), int64(60)).Return(slot, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return([]*model.ScheduleSlot{slot}, nil)
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(0, nil)
		m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
		m.trainings.EXPECT().
			CreateTraining(gomock.Any(), userID, friToday, model.TrainingTypeGi, int32(60), (*string)(nil)).
			Return(&model.Training{ID: 900, UserID: userID, Date: friToday, TrainingType: model.TrainingTypeGi, DurationMinutes: 60}, nil)
		m.bot.EXPECT().
			EditMessageWithKeyboard(gomock.Any(), chatID, messageID, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, _ int, text string, _ dto.Keyboard) error {
				assert.Contains(t, text, "Тренировка сохранена")

				return nil
			})

		in := dto.Input{
			ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "rec:60:20261016",
		}

		err := newUseCase(m, now).Handle(context.Background(), u, in)
		assert.NoError(t, err)
	})

	t.Run("a repeat tap shows the existing training instead of creating a duplicate", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.slots.EXPECT().GetSlot(gomock.Any(), int64(60)).Return(slot, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return([]*model.ScheduleSlot{slot}, nil)
		m.trainings.EXPECT().CountTrainingsOnDate(gomock.Any(), userID, friToday).Return(1, nil)
		m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
		m.trainings.EXPECT().
			LatestTrainingOnDate(gomock.Any(), userID, friToday).
			Return(&model.Training{ID: 900, UserID: userID, Date: friToday, TrainingType: model.TrainingTypeGi, DurationMinutes: 60}, nil)
		// No CreateTraining call.
		m.bot.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, messageID, gomock.Any(), gomock.Any()).Return(nil)

		in := dto.Input{
			ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "rec:60:20261016",
		}

		err := newUseCase(m, now).Handle(context.Background(), u, in)
		assert.NoError(t, err)
	})

	t.Run("a future date is rejected", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.slots.EXPECT().GetSlot(gomock.Any(), int64(60)).Return(slot, nil)
		m.bot.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", gomock.Any()).Return(nil)
		// No ListSlots/CreateTraining — rejected before reaching them.

		in := dto.Input{
			ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "rec:60:20261017",
		}

		err := newUseCase(m, now).Handle(context.Background(), u, in)
		assert.NoError(t, err)
	})

	t.Run("a date older than 3 days is rejected", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.slots.EXPECT().GetSlot(gomock.Any(), int64(60)).Return(slot, nil)
		m.bot.EXPECT().AnswerCallbackWithText(gomock.Any(), "cb-1", gomock.Any()).Return(nil)

		in := dto.Input{
			ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "rec:60:20261010",
		}

		err := newUseCase(m, now).Handle(context.Background(), u, in)
		assert.NoError(t, err)
	})

	t.Run("a slot belonging to someone else is rejected", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		other := *slot
		other.UserID = userID + 1
		m.slots.EXPECT().GetSlot(gomock.Any(), int64(60)).Return(&other, nil)
		m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

		in := dto.Input{
			ChatID: chatID, MessageID: messageID, HasCallback: true, CallbackID: "cb-1", CallbackData: "rec:60:20261016",
		}

		err := newUseCase(m, now).Handle(context.Background(), u, in)
		assert.NoError(t, err)
	})

	t.Run("a malformed callback is rejected", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "rec:notanumber:bogus"}

		err := newUseCase(m, now).Handle(context.Background(), u, in)
		assert.NoError(t, err)
	})

	t.Run("an unknown callback is just acknowledged", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"}

		err := newUseCase(m, now).Handle(context.Background(), u, in)
		assert.NoError(t, err)
	})

	t.Run("no callback — no-op", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		err := newUseCase(m, now).Handle(context.Background(), u, dto.Input{ChatID: chatID})
		assert.NoError(t, err)
	})
}
