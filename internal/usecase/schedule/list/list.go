// Package list owns the "📅 Расписание" screen — the recurring weekly
// slots a user has set, plus the entry point into schedule/create. Tapping
// a slot hands off to schedule/info via its own schedule:view:{id}
// callback, exactly how training/history hands off to training/info. It
// also owns the reminder toggle shown once at least one slot exists, and
// the four-zone timezone picker that backs it (see internal/usecase/reminder
// for the job this toggle feeds).
package list

import (
	"context"
	"fmt"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/timezone"
)

const (
	callbackScheduleList = "schedule:list"
	callbackScheduleAdd  = "schedule:add"
)

// callbackMenuBack mirrors other scenarios' own private constant.
const callbackMenuBack = "menu:back"

const (
	callbackReminderToggle = "schedule:reminders:toggle"
	callbackTZSetPrefix    = "schedule:tz:set:"
	callbackTZCancel       = "schedule:tz:cancel"
)

type UseCase struct {
	bot   sender
	repo  slotRepo
	users userRepo
}

func New(bot sender, repo slotRepo, users userRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo, users: users}
}

func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	switch {
	case in.CallbackData == callbackScheduleList:
		return uc.showList(ctx, u, in)

	case in.CallbackData == callbackReminderToggle:
		return uc.toggleReminders(ctx, u, in)

	case strings.HasPrefix(in.CallbackData, callbackTZSetPrefix):
		return uc.setTimezone(ctx, u, in)

	case in.CallbackData == callbackTZCancel:
		return uc.showList(ctx, u, in)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

func (uc *UseCase) showList(ctx context.Context, u *model.User, in dto.Input) error {
	slots, err := uc.repo.ListSlots(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("list schedule slots: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, listText(slots, u), listKeyboard(slots, u))
}

// toggleReminders flips RemindersEnabled, except when turning it on for the
// first time with no timezone set yet — then it shows the zone picker
// instead and leaves RemindersEnabled untouched until a zone is actually
// picked (see setTimezone).
func (uc *UseCase) toggleReminders(ctx context.Context, u *model.User, in dto.Input) error {
	if u.RemindersEnabled {
		u.RemindersEnabled = false

		if err := uc.users.Update(ctx, u); err != nil {
			return fmt.Errorf("update user: %w", err)
		}

		return uc.showList(ctx, u, in)
	}

	if u.Timezone == nil {
		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, tzPickerText, tzPickerKeyboard())
	}

	u.RemindersEnabled = true

	if err := uc.users.Update(ctx, u); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return uc.showList(ctx, u, in)
}

// setTimezone handles a tap on one of the picker's four zone buttons.
// Safe to run twice (e.g. a stale message tapped again after the flow
// already completed) — it only ever writes the same deterministic result.
func (uc *UseCase) setTimezone(ctx context.Context, u *model.User, in dto.Input) error {
	z, ok := timezone.ByCode(strings.TrimPrefix(in.CallbackData, callbackTZSetPrefix))
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	iana := z.IANA
	u.Timezone = &iana
	u.RemindersEnabled = true

	if err := uc.users.Update(ctx, u); err != nil {
		return fmt.Errorf("update user: %w", err)
	}

	return uc.showList(ctx, u, in)
}
