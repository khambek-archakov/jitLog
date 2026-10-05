// Package profile owns the "👤 Профиль" screen, its "✏️ Редактировать"
// submenu, the belt-change flow (callback-driven) and the age-change flow
// (the one field whose new value arrives as free text, backed by
// profile_edit_draft — see Continue).
package profile

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	callbackProfileShow          = "profile:show"
	callbackProfileEdit          = "profile:edit"
	callbackProfileBeltEdit      = "profile:belt:edit"
	callbackProfileBeltSetPrefix = "profile:belt:set:"
	callbackProfileAgeEdit       = "profile:age:edit"
)

type UseCase struct {
	bot    sender
	user   user
	drafts profileEditDraft
}

func New(bot sender, user user, drafts profileEditDraft) *UseCase {
	return &UseCase{bot: bot, user: user, drafts: drafts}
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

	return uc.bot.Send(ctx, in.ChatID, askAgeText)
}

func (uc *UseCase) setBelt(ctx context.Context, u *model.User, in dto.Input) error {
	belt, ok := beltFromToken(strings.TrimPrefix(in.CallbackData, callbackProfileBeltSetPrefix))
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	u.Belt = belt

	if err := uc.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user belt: %w", err)
	}

	err := uc.user.AddBeltPromotion(ctx, u.ID, belt, time.Now())
	if err != nil && !errors.Is(err, model.ErrDuplicateBeltPromotion) {
		return fmt.Errorf("add belt promotion: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, profileText(u), profileKeyboard())
}
