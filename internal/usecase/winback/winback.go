// Package winback owns the "haven't seen you in a while" nudge: a
// ticker-driven job (Run, called from the same 15-minute ticker as
// internal/usecase/reminder — see cmd/service/main.go) that sends a single
// animation (or, failing that, plain text) to a user who's gone quiet for
// 14+ days, plus the two buttons on that message's own callback (Handle).
package winback

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/menu"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/usertime"
)

const (
	// silenceThreshold is how long a user must have been quiet before a
	// nudge is even considered.
	silenceThreshold = 14 * 24 * time.Hour
	// resendThreshold is how long must pass after a previous nudge
	// (that the user did return from) before a second one is allowed.
	resendThreshold = 60 * 24 * time.Hour
	// recentReminderWindow — skip a user the weekly training reminder
	// already reached this recently; piling a nudge on top is noise.
	recentReminderWindow = 24 * time.Hour
	// windowStart/windowEnd bound the user's own local hours a nudge may
	// send in — never at night, regardless of time zone.
	windowStartHour = 10
	windowEndHour   = 20
)

type UseCase struct {
	bot             sender
	users           userRepo
	reminders       reminderRepo
	logger          *slog.Logger
	animationFileID string
	now             func() time.Time
}

func New(bot sender, users userRepo, reminders reminderRepo, logger *slog.Logger, animationFileID string) *UseCase {
	return NewWithClock(bot, users, reminders, logger, animationFileID, time.Now)
}

// NewWithClock lets tests substitute a fixed instant instead of the real
// wall clock — see internal/usecase/reminder's identical pattern.
func NewWithClock(
	bot sender, users userRepo, reminders reminderRepo, logger *slog.Logger, animationFileID string,
	now func() time.Time,
) *UseCase {
	return &UseCase{
		bot: bot, users: users, reminders: reminders, logger: logger, animationFileID: animationFileID, now: now,
	}
}

// Run walks every user and sends at most one nudge per user per tick. One
// user's failure is logged and skipped rather than aborting the rest of
// the walk — same reasoning as internal/usecase/reminder.Run.
func (uc *UseCase) Run(ctx context.Context) error {
	users, err := uc.users.ListAll(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}

	for _, u := range users {
		if err := uc.processUser(ctx, u); err != nil {
			uc.logger.Error("winback: failed to process user", "user_id", u.ID, "error", err)
		}
	}

	return nil
}

func (uc *UseCase) processUser(ctx context.Context, u *model.User) error {
	now := uc.now()

	if u.WinbackDeclinedAt != nil && !u.WinbackDeclinedAt.Before(u.LastActivityAt) {
		return nil // declined, and nothing's happened since — stays declined.
	}

	if now.Sub(u.LastActivityAt) < silenceThreshold {
		return nil // not quiet long enough yet.
	}

	if u.WinbackSentAt != nil {
		if !u.WinbackSentAt.Before(u.LastActivityAt) {
			return nil // already sent once for this exact silence — never a second.
		}

		if now.Sub(*u.WinbackSentAt) < resendThreshold {
			return nil // came back since, but not long enough ago to nudge again.
		}
	}

	loc := usertime.Location(u)
	hour := now.In(loc).Hour()

	if hour < windowStartHour || hour >= windowEndHour {
		return nil // outside this user's own local daytime window.
	}

	recentReminder, err := uc.reminders.HasSentWithin(ctx, u.ID, now.Add(-recentReminderWindow))
	if err != nil {
		return fmt.Errorf("check recent weekly reminder: %w", err)
	}

	if recentReminder {
		return nil
	}

	if err := uc.send(ctx, u); err != nil {
		if errors.Is(err, model.ErrUserBlockedBot) {
			if upErr := uc.users.SetWinbackDeclined(ctx, u.ID, now); upErr != nil {
				return fmt.Errorf("decline for blocked user: %w", upErr)
			}

			uc.logger.Info("winback: user blocked the bot, declined", "user_id", u.ID)

			return nil
		}

		return fmt.Errorf("send winback message: %w", err)
	}

	if err := uc.users.SetWinbackSent(ctx, u.ID, now); err != nil {
		return fmt.Errorf("set winback sent: %w", err)
	}

	return nil
}

// send tries the configured animation first, falling back to plain text
// when no file ID is configured or the animation send itself fails for a
// reason other than the user having blocked the bot (in which case a text
// fallback would fail identically, so there's no point trying it). The
// animation itself carries no caption — only the text fallback shows
// greetingText.
func (uc *UseCase) send(ctx context.Context, u *model.User) error {
	if uc.animationFileID == "" {
		uc.logger.Info("winback: no animation file id configured, sending plain text", "user_id", u.ID)

		return uc.bot.SendWithKeyboard(ctx, u.TelegramID, greetingText, winbackKeyboard())
	}

	err := uc.bot.SendAnimationWithKeyboard(ctx, u.TelegramID, uc.animationFileID, "", winbackKeyboard())
	if err == nil {
		return nil
	}

	if errors.Is(err, model.ErrUserBlockedBot) {
		return err
	}

	uc.logger.Warn("winback: animation send failed, falling back to plain text", "user_id", u.ID, "error", err)

	return uc.bot.SendWithKeyboard(ctx, u.TelegramID, greetingText, winbackKeyboard())
}

// Handle covers the two buttons on whatever winback message Run sent —
// both strip that message's own keyboard (best-effort: safe to repeat
// even on an already-bare or long-stale message) and then show the main
// menu as a new message; winback:off additionally records the decline.
func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	if in.CallbackData != callbackHi && in.CallbackData != callbackOff {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	if err := uc.bot.RemoveKeyboard(ctx, in.ChatID, in.MessageID); err != nil {
		uc.logger.Warn("winback: failed to remove keyboard", "error", err)
	}

	text := menu.Text

	if in.CallbackData == callbackOff {
		if err := uc.users.SetWinbackDeclined(ctx, u.ID, uc.now()); err != nil {
			return fmt.Errorf("set winback declined: %w", err)
		}

		text = declinedText + "\n\n" + text
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, text, menu.Keyboard())
}
