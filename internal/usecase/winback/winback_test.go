package winback_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/menu"
	"github.com/khambek-archakov/jitLog/internal/usecase/winback"
)

const (
	userID, telegramID int64 = 1, 999
	fileID                   = "anim-file-id"
)

// now is Friday 14:00 UTC == 17:00 Moscow (+3) — inside the 10:00-20:00
// local window every test that doesn't specifically probe the window
// boundary relies on.
var now = time.Date(2026, 10, 16, 14, 0, 0, 0, time.UTC)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func ptr[T any](v T) *T {
	return &v
}

func moscowUser(lastActivityAt time.Time) *model.User {
	return &model.User{
		ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Moscow"), LastActivityAt: lastActivityAt,
	}
}

type mocks struct {
	bot       *Mocksender
	users     *MockuserRepo
	reminders *MockreminderRepo
}

func newMocks(ctrl *gomock.Controller) mocks {
	return mocks{bot: NewMocksender(ctrl), users: NewMockuserRepo(ctrl), reminders: NewMockreminderRepo(ctrl)}
}

func newUseCase(m mocks, clock time.Time) *winback.UseCase {
	return winback.NewWithClock(m.bot, m.users, m.reminders, discardLogger(), fileID, func() time.Time { return clock })
}

func newUseCaseNoAnimation(m mocks, clock time.Time) *winback.UseCase {
	return winback.NewWithClock(m.bot, m.users, m.reminders, discardLogger(), "", func() time.Time { return clock })
}

// expectEligibleSend wires the common "this user qualifies, send and mark
// sent" mock expectations.
func expectEligibleSend(m mocks) {
	m.reminders.EXPECT().HasSentWithin(gomock.Any(), userID, now.Add(-24*time.Hour)).Return(false, nil)
	m.bot.EXPECT().SendAnimationWithKeyboard(gomock.Any(), telegramID, fileID, "", gomock.Any()).Return(nil)
	m.users.EXPECT().SetWinbackSent(gomock.Any(), userID, now).Return(nil)
}

func TestRun(t *testing.T) {
	t.Parallel()

	t.Run("13 days of silence — not sent", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-13 * 24 * time.Hour))
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		// No other calls — not silent long enough even to check the window.

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("15 days of silence — sent", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-15 * 24 * time.Hour))
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		expectEligibleSend(m)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("local time window", func(t *testing.T) {
		t.Parallel()

		t.Run("Europe/Kaliningrad (+2) outside 10:00-20:00 — not sent", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMocks(ctrl)

			// 06:00 UTC == 08:00 Kaliningrad — before the window opens.
			clock := time.Date(2026, 10, 16, 6, 0, 0, 0, time.UTC)

			u := &model.User{
				ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Kaliningrad"),
				LastActivityAt: clock.Add(-15 * 24 * time.Hour),
			}
			m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)

			err := newUseCase(m, clock).Run(context.Background())
			assert.NoError(t, err)
		})

		t.Run("Europe/Kaliningrad (+2) inside 10:00-20:00 — sent", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMocks(ctrl)

			// 13:00 UTC == 15:00 Kaliningrad — inside the window.
			clock := time.Date(2026, 10, 16, 13, 0, 0, 0, time.UTC)

			u := &model.User{
				ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Kaliningrad"),
				LastActivityAt: clock.Add(-15 * 24 * time.Hour),
			}
			m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
			m.reminders.EXPECT().HasSentWithin(gomock.Any(), userID, clock.Add(-24*time.Hour)).Return(false, nil)
			m.bot.EXPECT().SendAnimationWithKeyboard(gomock.Any(), telegramID, fileID, "", gomock.Any()).Return(nil)
			m.users.EXPECT().SetWinbackSent(gomock.Any(), userID, clock).Return(nil)

			err := newUseCase(m, clock).Run(context.Background())
			assert.NoError(t, err)
		})

		t.Run("Asia/Yekaterinburg (+5) outside 10:00-20:00 — not sent", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMocks(ctrl)

			// 16:00 UTC == 21:00 Yekaterinburg — past the window's close.
			clock := time.Date(2026, 10, 16, 16, 0, 0, 0, time.UTC)

			u := &model.User{
				ID: userID, TelegramID: telegramID, Timezone: ptr("Asia/Yekaterinburg"),
				LastActivityAt: clock.Add(-15 * 24 * time.Hour),
			}
			m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)

			err := newUseCase(m, clock).Run(context.Background())
			assert.NoError(t, err)
		})

		t.Run("Asia/Yekaterinburg (+5) inside 10:00-20:00 — sent", func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			m := newMocks(ctrl)

			// 10:00 UTC == 15:00 Yekaterinburg — inside the window.
			clock := time.Date(2026, 10, 16, 10, 0, 0, 0, time.UTC)

			u := &model.User{
				ID: userID, TelegramID: telegramID, Timezone: ptr("Asia/Yekaterinburg"),
				LastActivityAt: clock.Add(-15 * 24 * time.Hour),
			}
			m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
			m.reminders.EXPECT().HasSentWithin(gomock.Any(), userID, clock.Add(-24*time.Hour)).Return(false, nil)
			m.bot.EXPECT().SendAnimationWithKeyboard(gomock.Any(), telegramID, fileID, "", gomock.Any()).Return(nil)
			m.users.EXPECT().SetWinbackSent(gomock.Any(), userID, clock).Return(nil)

			err := newUseCase(m, clock).Run(context.Background())
			assert.NoError(t, err)
		})
	})

	t.Run("a second run in the same silence sends no duplicate", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		lastActivity := now.Add(-15 * 24 * time.Hour)
		sentAt := now.Add(-time.Hour) // sent earlier today, since this same silence began

		u := &model.User{
			ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Moscow"),
			LastActivityAt: lastActivity, WinbackSentAt: &sentAt,
		}
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		// sentAt is not before lastActivity — no further calls at all.

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("returned, quiet 14 days again, but under 60 days since the previous send — not sent", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		sentAt := now.Add(-40 * 24 * time.Hour)       // previous nudge, 40 days ago
		lastActivity := now.Add(-15 * 24 * time.Hour) // returned after that, then quiet 15 days

		u := &model.User{
			ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Moscow"),
			LastActivityAt: lastActivity, WinbackSentAt: &sentAt,
		}
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("returned, quiet 14 days again, over 60 days since the previous send — sent", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		sentAt := now.Add(-70 * 24 * time.Hour)
		lastActivity := now.Add(-15 * 24 * time.Hour)

		u := &model.User{
			ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Moscow"),
			LastActivityAt: lastActivity, WinbackSentAt: &sentAt,
		}
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		expectEligibleSend(m)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("declined and nothing since — never sent, at 15, 60 or 200 days", func(t *testing.T) {
		t.Parallel()

		for _, days := range []int{15, 60, 200} {
			t.Run(fmt.Sprintf("%d days", days), func(t *testing.T) {
				t.Parallel()

				ctrl := gomock.NewController(t)
				m := newMocks(ctrl)

				at := now.Add(-time.Duration(days) * 24 * time.Hour)

				u := &model.User{
					ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Moscow"),
					LastActivityAt: at, WinbackDeclinedAt: &at, // declined_at == last_activity_at — still active
				}
				m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
				// No other calls — declined gate stops it immediately.

				err := newUseCase(m, now).Run(context.Background())
				assert.NoError(t, err)
			})
		}
	})

	t.Run("declined, then any action, then quiet 14 days with the 60-day send gate met — sent", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		sentAt := now.Add(-90 * 24 * time.Hour)
		declinedAt := now.Add(-50 * 24 * time.Hour)
		lastActivity := now.Add(-20 * 24 * time.Hour) // the action that lifted the decline, then quiet since

		u := &model.User{
			ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Moscow"),
			LastActivityAt: lastActivity, WinbackSentAt: &sentAt, WinbackDeclinedAt: &declinedAt,
		}
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		expectEligibleSend(m)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("the weekly training reminder landed less than 24h ago — not sent", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-15 * 24 * time.Hour))
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		m.reminders.EXPECT().HasSentWithin(gomock.Any(), userID, now.Add(-24*time.Hour)).Return(true, nil)
		// No bot call.

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a blocked user has winback declined and isn't retried this run", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-15 * 24 * time.Hour))
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		m.reminders.EXPECT().HasSentWithin(gomock.Any(), userID, now.Add(-24*time.Hour)).Return(false, nil)
		m.bot.EXPECT().
			SendAnimationWithKeyboard(gomock.Any(), telegramID, fileID, "", gomock.Any()).
			Return(model.ErrUserBlockedBot)
		m.users.EXPECT().SetWinbackDeclined(gomock.Any(), userID, now).Return(nil)
		// No text fallback, no SetWinbackSent.

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("after a blocked user returns (any action lifts the decline), they're eligible again", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		declinedAt := now.Add(-30 * 24 * time.Hour)
		lastActivity := now.Add(-15 * 24 * time.Hour) // acted after declining — decline lifted, then quiet 15 days

		u := &model.User{
			ID: userID, TelegramID: telegramID, Timezone: ptr("Europe/Moscow"),
			LastActivityAt: lastActivity, WinbackDeclinedAt: &declinedAt,
		}
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		expectEligibleSend(m)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("a just-migrated user (last_activity_at ~ now) gets nothing on the first run", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-time.Minute))
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		// No other calls.

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("failed to list users", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.users.EXPECT().ListAll(gomock.Any()).Return(nil, errors.New("fail"))

		err := newUseCase(m, now).Run(context.Background())
		assert.Error(t, err)
	})

	t.Run("one user's failure is logged and doesn't stop the rest of the walk", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		broken := moscowUser(now.Add(-15 * 24 * time.Hour))
		broken.ID = 2

		ok := moscowUser(now.Add(-15 * 24 * time.Hour))

		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{broken, ok}, nil)
		m.reminders.EXPECT().HasSentWithin(gomock.Any(), int64(2), gomock.Any()).Return(false, errors.New("fail"))
		expectEligibleSend(m)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})
}

func TestSend_AnimationFallback(t *testing.T) {
	t.Parallel()

	t.Run("no animation file id configured — sends plain text directly", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-15 * 24 * time.Hour))
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		m.reminders.EXPECT().HasSentWithin(gomock.Any(), userID, gomock.Any()).Return(false, nil)
		m.bot.EXPECT().
			SendWithKeyboard(gomock.Any(), telegramID, "Привет 👋", gomock.Any()).
			DoAndReturn(func(_ context.Context, _ int64, _ string, kb dto.Keyboard) error {
				require.Len(t, kb, 2)
				require.Len(t, kb[0], 1)
				require.Len(t, kb[1], 1)
				assert.Equal(t, "winback:hi", kb[0][0].Data)
				assert.Equal(t, "winback:off", kb[1][0].Data)

				return nil
			})
		m.users.EXPECT().SetWinbackSent(gomock.Any(), userID, now).Return(nil)
		// No SendAnimationWithKeyboard call at all.

		err := newUseCaseNoAnimation(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("animation send fails for a non-block reason — falls back to text, still marks sent", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-15 * 24 * time.Hour))
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		m.reminders.EXPECT().HasSentWithin(gomock.Any(), userID, gomock.Any()).Return(false, nil)
		m.bot.EXPECT().
			SendAnimationWithKeyboard(gomock.Any(), telegramID, fileID, "", gomock.Any()).
			Return(errors.New("bad file id"))
		m.bot.EXPECT().SendWithKeyboard(gomock.Any(), telegramID, "Привет 👋", gomock.Any()).Return(nil)
		m.users.EXPECT().SetWinbackSent(gomock.Any(), userID, now).Return(nil)

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})

	t.Run("animation fails because the bot is blocked — no text fallback attempted", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		u := moscowUser(now.Add(-15 * 24 * time.Hour))
		m.users.EXPECT().ListAll(gomock.Any()).Return([]*model.User{u}, nil)
		m.reminders.EXPECT().HasSentWithin(gomock.Any(), userID, gomock.Any()).Return(false, nil)
		m.bot.EXPECT().
			SendAnimationWithKeyboard(gomock.Any(), telegramID, fileID, "", gomock.Any()).
			Return(model.ErrUserBlockedBot)
		m.users.EXPECT().SetWinbackDeclined(gomock.Any(), userID, now).Return(nil)
		// No SendWithKeyboard fallback call.

		err := newUseCase(m, now).Run(context.Background())
		assert.NoError(t, err)
	})
}

func TestHandle(t *testing.T) {
	t.Parallel()

	const chatID, messageID int64 = 777, 555

	u := &model.User{ID: userID, TelegramID: telegramID}

	t.Run("winback:hi removes the keyboard and shows the menu", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
		m.bot.EXPECT().RemoveKeyboard(gomock.Any(), chatID, int(messageID)).Return(nil)
		m.bot.EXPECT().SendWithKeyboard(gomock.Any(), chatID, menu.Text, gomock.Any()).Return(nil)
		// No SetWinbackDeclined call.

		in := dto.Input{
			ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "winback:hi",
		}

		err := newUseCase(m, now).Handle(context.Background(), u, in)
		assert.NoError(t, err)
	})

	t.Run("winback:off removes the keyboard, declines, shows the standard menu", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
		m.bot.EXPECT().RemoveKeyboard(gomock.Any(), chatID, int(messageID)).Return(nil)
		m.users.EXPECT().SetWinbackDeclined(gomock.Any(), userID, now).Return(nil)
		m.bot.EXPECT().SendWithKeyboard(gomock.Any(), chatID, menu.Text, menu.Keyboard()).Return(nil)

		in := dto.Input{
			ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "winback:off",
		}

		err := newUseCase(m, now).Handle(context.Background(), u, in)
		assert.NoError(t, err)
	})

	t.Run("a failed RemoveKeyboard is logged but doesn't block showing the menu — safe repeat tap", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
		m.bot.EXPECT().RemoveKeyboard(gomock.Any(), chatID, int(messageID)).Return(errors.New("message is not modified"))
		m.bot.EXPECT().SendWithKeyboard(gomock.Any(), chatID, menu.Text, gomock.Any()).Return(nil)

		in := dto.Input{
			ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "winback:hi",
		}

		err := newUseCase(m, now).Handle(context.Background(), u, in)
		assert.NoError(t, err)
	})

	t.Run("failing to record the decline propagates as an error", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		m := newMocks(ctrl)

		m.bot.EXPECT().AnswerCallback(gomock.Any(), "cb-1").Return(nil)
		m.bot.EXPECT().RemoveKeyboard(gomock.Any(), chatID, int(messageID)).Return(nil)
		m.users.EXPECT().SetWinbackDeclined(gomock.Any(), userID, now).Return(errors.New("fail"))

		in := dto.Input{
			ChatID: chatID, MessageID: int(messageID), HasCallback: true, CallbackID: "cb-1", CallbackData: "winback:off",
		}

		err := newUseCase(m, now).Handle(context.Background(), u, in)
		assert.Error(t, err)
	})

	t.Run("unknown callback is just acknowledged", func(t *testing.T) {
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
