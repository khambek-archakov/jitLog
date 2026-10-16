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

// friSlot is a single Friday-evening slot — the common-case fixture most
// tests build on. 2026-10-16 is a Friday, 2026-10-12 its Monday.
func friSlot() []*model.ScheduleSlot {
	return []*model.ScheduleSlot{{DayOfWeek: 5, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeGi}}
}

var (
	friWeekStart = time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	friToday     = time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)
)

const userID, telegramID int64 = 1, 999

func moscowUser() *model.User {
	return &model.User{ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Moscow"), RemindersEnabled: true}
}

type mocks struct {
	bot       *Mocksender
	users     *MockuserRepo
	slots     *MockslotRepo
	trainings *MocktrainingRepo
	sent      *MocksentRepo
}

func newUseCase(m mocks, now time.Time) *reminder.UseCase {
	return reminder.NewWithClock(m.bot, m.users, m.slots, m.trainings, m.sent, discardLogger(), func() time.Time { return now })
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

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("sends inside the window, no prior training, not yet sent", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser()
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, friWeekStart).Return(false, nil)

		m.bot.EXPECT().
			SendWithKeyboard(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, text string, kb dto.Keyboard) error {
				assert.Contains(t, text, "На этой неделе ещё не было тренировок")
				assert.Contains(t, text, "19:00")
				require.Len(t, kb, 1)
				assert.Equal(t, "reminder:disable", kb[0][0].Data)

				return nil
			})

		m.sent.EXPECT().MarkSent(gomock.Any(), userID, friWeekStart).Return(nil)

		// Friday 17:00 Moscow (+3) == 14:00 UTC — inside [16:00, 19:00).
		now := time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("target before 09:00 clamps to 09:00, still sends inside the clamped window", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser()
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().
			ListSlots(gomock.Any(), userID).
			Return([]*model.ScheduleSlot{{DayOfWeek: 5, TimeMinutes: 10 * 60, TrainingType: model.TrainingTypeGi}}, nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, friWeekStart).Return(false, nil)
		m.bot.EXPECT().SendWithKeyboard(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).Return(nil)
		m.sent.EXPECT().MarkSent(gomock.Any(), userID, friWeekStart).Return(nil)

		// Slot at 10:00 -> raw target 07:00, clamped to 09:00. Now = 09:30
		// Moscow (+3) == 06:30 UTC — inside [09:00, 10:00).
		now := time.Date(2026, 10, 16, 6, 30, 0, 0, time.UTC)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("clamped target reaching the slot itself means no reminder, ever, that day", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser()
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().
			ListSlots(gomock.Any(), userID).
			Return([]*model.ScheduleSlot{{DayOfWeek: 5, TimeMinutes: 8 * 60, TrainingType: model.TrainingTypeGi}}, nil)
		// No other mock calls expected — processUser must bail out right
		// after computing the clamped window, before touching training/sent/bot.

		// Slot at 08:00 -> raw target 05:00, clamped to 09:00, which is no
		// longer before the slot (08:00) — the window is empty.
		now := time.Date(2026, 10, 16, 4, 30, 0, 0, time.UTC) // 07:30 Moscow

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("already trained this week — no reminder", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser()
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(true, nil)
		// No sent/bot calls expected at all.

		now := time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("already sent this week — re-running in the same window sends no duplicate", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser()
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, friWeekStart).Return(true, nil)
		// No bot.SendWithKeyboard/MarkSent expected.

		now := time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a slot only on Monday fires on Monday", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser()
		mondayWeekStart := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)

		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().
			ListSlots(gomock.Any(), userID).
			Return([]*model.ScheduleSlot{{DayOfWeek: 1, TimeMinutes: 19 * 60, TrainingType: model.TrainingTypeNoGi}}, nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, mondayWeekStart, mondayWeekStart).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, mondayWeekStart).Return(false, nil)
		m.bot.EXPECT().SendWithKeyboard(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).Return(nil)
		m.sent.EXPECT().MarkSent(gomock.Any(), userID, mondayWeekStart).Return(nil)

		// Monday 17:00 Moscow == 14:00 UTC.
		now := time.Date(2026, 10, 19, 14, 0, 0, 0, time.UTC)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("no schedule at all — no reminder", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser()
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(nil, nil)

		now := time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("today isn't the last scheduled day — no reminder", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser()
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil) // last day is Friday

		// Wednesday 17:00 Moscow == 14:00 UTC — not Friday.
		now := time.Date(2026, 10, 14, 14, 0, 0, 0, time.UTC)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("nil timezone is skipped outright, never falls back to Moscow for this job", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := &model.User{ID: userID, TelegramID: telegramID, RemindersEnabled: true}
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		// No slots/trainings/sent/bot calls at all — bails out before
		// even listing the schedule.

		now := time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a blocked user has reminders disabled and is not retried this run", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser()
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, friWeekStart).Return(false, nil)
		m.bot.EXPECT().
			SendWithKeyboard(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).
			Return(model.ErrUserBlockedBot)
		m.users.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, u *model.User) error {
				assert.False(t, u.RemindersEnabled)

				return nil
			})
		// No MarkSent — a blocked send never counts as sent.

		now := time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("any other send error is logged only — no retry, no mark-sent, run still succeeds", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser()
		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().ListSlots(gomock.Any(), userID).Return(friSlot(), nil)
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), userID, friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), userID, friWeekStart).Return(false, nil)
		m.bot.EXPECT().
			SendWithKeyboard(gomock.Any(), telegramID, gomock.Any(), gomock.Any()).
			Return(errors.New("telegram is down"))
		// No Update, no MarkSent.

		now := time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a user in a different zone gets their own zone-correct week_start", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := &model.User{ID: 2, TelegramID: 2002, Timezone: ptr("Asia/Yekaterinburg"), RemindersEnabled: true}

		m.users.EXPECT().ListRemindersEnabled(gomock.Any()).Return([]*model.User{u}, nil)
		m.slots.EXPECT().
			ListSlots(gomock.Any(), int64(2)).
			Return([]*model.ScheduleSlot{{DayOfWeek: 5, TimeMinutes: 19 * 60}}, nil)

		// 2026-10-16 is also a Friday in Yekaterinburg (+5): same calendar
		// week as the Moscow fixture above, computed independently off this
		// user's own zone rather than any shared/default one.
		m.trainings.EXPECT().HasTrainingBetween(gomock.Any(), int64(2), friWeekStart, friToday).Return(false, nil)
		m.sent.EXPECT().HasSent(gomock.Any(), int64(2), friWeekStart).Return(false, nil)
		m.bot.EXPECT().SendWithKeyboard(gomock.Any(), int64(2002), gomock.Any(), gomock.Any()).Return(nil)
		m.sent.EXPECT().MarkSent(gomock.Any(), int64(2), friWeekStart).Return(nil)

		// Friday 17:00 Yekaterinburg (+5) == 12:00 UTC — inside [16:00, 19:00).
		now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})
}

func TestHandle(t *testing.T) {
	t.Parallel()

	const chatID, messageID int64 = 777, 555

	t.Run("reminder:disable turns reminders off", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser()

		m.users.EXPECT().
			Update(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, u *model.User) error {
				assert.False(t, u.RemindersEnabled)

				return nil
			})
		m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
		m.bot.EXPECT().EditMessageWithKeyboard(gomock.Any(), chatID, int(messageID), gomock.Any(), gomock.Any()).Return(nil)

		uc := newUseCase(m, time.Now())

		in := dto.Input{
			ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "reminder:disable",
		}

		err := uc.Handle(context.Background(), u, in)
		assert.NoError(t, err)
	})

	t.Run("unknown callback is just acknowledged", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)

		uc := newUseCase(m, time.Now())

		in := dto.Input{ChatID: chatID, HasCallback: true, CallbackID: "cb-1", CallbackData: "junk"}

		err := uc.Handle(context.Background(), moscowUser(), in)
		assert.NoError(t, err)
	})

	t.Run("no callback — no-op", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		uc := newUseCase(m, time.Now())

		err := uc.Handle(context.Background(), moscowUser(), dto.Input{ChatID: chatID})
		assert.NoError(t, err)
	})
}
