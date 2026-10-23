// Package profile owns the "👤 Профиль" screen, its "✏️ Редактировать"
// submenu, the belt-change flow (callback-driven) and the age-change flow
// (the one field whose new value arrives as free text, backed by
// profile_edit_draft — see Continue).
package profile

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/timezone"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/usertime"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/stats"
)

const (
	callbackProfileShow          = "profile:show"
	callbackProfileEdit          = "profile:edit"
	callbackProfileBeltEdit      = "profile:belt:edit"
	callbackProfileBeltSetPrefix = "profile:belt:set:"
	callbackProfileAgeEdit       = "profile:age:edit"
	callbackProfileAgeCancel     = "profile:age:cancel"
	callbackProfileTZEdit        = "profile:tz:edit"
	callbackProfileTZSetPrefix   = "profile:tz:set:"
)

type UseCase struct {
	bot       sender
	user      user
	drafts    profileEditDraft
	trainings trainingRepo
	logger    *slog.Logger
}

func New(bot sender, user user, drafts profileEditDraft, trainings trainingRepo, logger *slog.Logger) *UseCase {
	return &UseCase{bot: bot, user: user, drafts: drafts, trainings: trainings, logger: logger}
}

func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	switch {
	case in.CallbackData == callbackProfileShow:
		return uc.showProfile(ctx, u, in)

	case in.CallbackData == callbackProfileEdit:
		return uc.showEditMenu(ctx, u, in)

	case in.CallbackData == callbackProfileBeltEdit:
		return uc.showBeltPicker(ctx, u, in)

	case strings.HasPrefix(in.CallbackData, callbackProfileBeltSetPrefix):
		return uc.setBelt(ctx, u, in)

	case in.CallbackData == callbackProfileAgeEdit:
		return uc.beginAgeEdit(ctx, u, in)

	case in.CallbackData == callbackProfileAgeCancel:
		return uc.cancelAgeEdit(ctx, u, in)

	case in.CallbackData == callbackProfileTZEdit:
		return uc.showTZPicker(ctx, u, in)

	case strings.HasPrefix(in.CallbackData, callbackProfileTZSetPrefix):
		return uc.setTZ(ctx, u, in)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

// Continue applies a free-text reply as the user's new age — the only
// path where Router hands us a pending model.ProfileEditDraft instead of
// a callback.
func (uc *UseCase) Continue(ctx context.Context, u *model.User, d *model.ProfileEditDraft, in dto.Input) error {
	if !in.HasMessage {
		return nil
	}

	age, ok := parseAge(in.Text)
	if !ok {
		return uc.bot.Send(ctx, in.ChatID, ageNotParsedText)
	}

	u.Age = &age

	if err := uc.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user age: %w", err)
	}

	if err := uc.drafts.DeleteEditDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete profile edit draft: %w", err)
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, profileText(u), profileKeyboard())
}

func (uc *UseCase) showProfile(ctx context.Context, u *model.User, in dto.Input) error {
	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, profileText(u), profileKeyboard())
}

func (uc *UseCase) showEditMenu(ctx context.Context, u *model.User, in dto.Input) error {
	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, editMenuText, editMenuKeyboard())
}

func (uc *UseCase) showBeltPicker(ctx context.Context, u *model.User, in dto.Input) error {
	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, beltPickerText, beltPickerKeyboard(u.Belt))
}

func (uc *UseCase) beginAgeEdit(ctx context.Context, u *model.User, in dto.Input) error {
	if err := uc.drafts.SetEditDraft(ctx, u.ID); err != nil {
		return fmt.Errorf("set profile edit draft: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, askAgeText, cancelAgeKeyboard())
}

// cancelAgeEdit backs out of the free-text age prompt without requiring any
// text at all, re-showing the edit submenu right in place of the prompt —
// the screen the "✏️ Возраст" button was tapped from.
func (uc *UseCase) cancelAgeEdit(ctx context.Context, u *model.User, in dto.Input) error {
	if err := uc.drafts.DeleteEditDraft(ctx, u.ID); err != nil {
		return fmt.Errorf("delete profile edit draft: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, editMenuText, editMenuKeyboard())
}

func (uc *UseCase) showTZPicker(ctx context.Context, u *model.User, in dto.Input) error {
	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, tzPickerText, tzPickerKeyboard())
}

// setTZ saves a newly picked zone without touching RemindersEnabled —
// changing timezone from here is a correction, not an opt-in/opt-out.
// Safe to run twice (a stale picker message tapped again).
func (uc *UseCase) setTZ(ctx context.Context, u *model.User, in dto.Input) error {
	z, ok := timezone.ByCode(strings.TrimPrefix(in.CallbackData, callbackProfileTZSetPrefix))
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	iana := z.IANA
	u.Timezone = &iana

	if err := uc.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user timezone: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, profileText(u), profileKeyboard())
}

func cancelAgeKeyboard() dto.Keyboard {
	return dto.Keyboard{dto.Row(dto.Button{Label: "❌ Отмена", Data: callbackProfileAgeCancel})}
}

// setBelt saves a belt change (user.belt and, only for a genuine step-up
// promotion, a belt_promotion row — see ChangeBelt) atomically, redraws
// the profile card, then — only once that's done — tries to send a
// congratulation as its own new message when the change is exactly one
// step up and the promotion row was actually fresh. A failure sending
// that congratulation is logged, never rolls back the belt change and
// never retries; see sendCongratulation.
func (uc *UseCase) setBelt(ctx context.Context, u *model.User, in dto.Input) error {
	newBelt, ok := beltFromToken(strings.TrimPrefix(in.CallbackData, callbackProfileBeltSetPrefix))
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	oldBelt := u.Belt
	promotedAt := usertime.Today(u)
	recordPromotion := beltRank(newBelt) > beltRank(oldBelt)

	recorded, err := uc.user.ChangeBelt(ctx, u.ID, newBelt, promotedAt, recordPromotion)
	if err != nil {
		return fmt.Errorf("change belt: %w", err)
	}

	u.Belt = newBelt

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	if err := uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, profileText(u), profileKeyboard()); err != nil {
		return err
	}

	if recorded && beltRank(newBelt) == beltRank(oldBelt)+1 {
		uc.sendCongratulation(ctx, u, oldBelt, newBelt, promotedAt, in.ChatID)
	}

	return nil
}

// sendCongratulation never returns an error — a failure anywhere in here
// (fetching the previous belt's own promoted_at, totaling its trainings,
// or the send itself) is logged and dropped, exactly as if the belt
// change had already fully succeeded without a congratulation, which it
// has: setBelt already committed and redrew the profile before this runs.
func (uc *UseCase) sendCongratulation(
	ctx context.Context, u *model.User, oldBelt, newBelt model.Belt, newPromotedAt time.Time, chatID int64,
) {
	text, err := uc.congratulationText(ctx, u, oldBelt, newBelt, newPromotedAt)
	if err != nil {
		uc.logger.Error("profile: failed to build belt congratulation", "user_id", u.ID, "error", err)

		return
	}

	if err := uc.bot.Send(ctx, chatID, text); err != nil {
		uc.logger.Error("profile: failed to send belt congratulation", "user_id", u.ID, "error", err)
	}
}

func (uc *UseCase) congratulationText(
	ctx context.Context, u *model.User, oldBelt, newBelt model.Belt, newPromotedAt time.Time,
) (string, error) {
	text := "🎉 Поздравляю с " + beltInstrumental(newBelt) + " поясом!"

	promotions, err := uc.user.ListBeltPromotions(ctx, u.ID)
	if err != nil {
		return "", fmt.Errorf("list belt promotions: %w", err)
	}

	oldPromotedAt, ok := promotedAtFor(promotions, oldBelt)
	if !ok {
		return text, nil
	}

	trainings, err := uc.trainings.ListTrainingsBetween(ctx, u.ID, oldPromotedAt, newPromotedAt)
	if err != nil {
		return "", fmt.Errorf("list trainings between: %w", err)
	}

	if len(trainings) == 0 {
		return text, nil
	}

	var totalMinutes int32
	for _, t := range trainings {
		totalMinutes += t.DurationMinutes
	}

	n := len(trainings)

	return text + fmt.Sprintf(
		"\nНа %s поясе в JitLog: %d %s, %s.",
		beltPrepositional(oldBelt), n, trainingWord(n), stats.FormatDuration(totalMinutes),
	), nil
}

func promotedAtFor(promotions []*model.BeltPromotion, belt model.Belt) (time.Time, bool) {
	for _, p := range promotions {
		if p.Belt == belt {
			return p.PromotedAt, true
		}
	}

	return time.Time{}, false
}
