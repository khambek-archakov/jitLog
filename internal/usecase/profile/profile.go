// Package profile owns the "👤 Профиль" screen — read-only user info plus
// the belt-change flow. Fully stateless: every step rides on callback data
// and the already-resolved *model.User, no draft table needed.
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
	callbackProfileBeltEdit      = "profile:belt:edit"
	callbackProfileBeltSetPrefix = "profile:belt:set:"
)

type UseCase struct {
	bot  sender
	user user
}

func New(bot sender, user user) *UseCase {
	return &UseCase{bot: bot, user: user}
}

func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	switch {
	case in.CallbackData == callbackProfileShow:
		return uc.showProfile(u, in)

	case in.CallbackData == callbackProfileBeltEdit:
		return uc.showBeltPicker(u, in)

	case strings.HasPrefix(in.CallbackData, callbackProfileBeltSetPrefix):
		return uc.setBelt(ctx, u, in)

	default:
		return uc.bot.AnswerCallback(in.CallbackID)
	}
}

func (uc *UseCase) showProfile(u *model.User, in dto.Input) error {
	if err := uc.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(in.ChatID, in.MessageID, profileText(u), profileKeyboard())
}

func (uc *UseCase) showBeltPicker(u *model.User, in dto.Input) error {
	if err := uc.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(in.ChatID, in.MessageID, beltPickerText, beltPickerKeyboard(u.Belt))
}

func (uc *UseCase) setBelt(ctx context.Context, u *model.User, in dto.Input) error {
	belt, ok := beltFromToken(strings.TrimPrefix(in.CallbackData, callbackProfileBeltSetPrefix))
	if !ok {
		return uc.bot.AnswerCallback(in.CallbackID)
	}

	u.Belt = belt

	if err := uc.user.Update(ctx, u); err != nil {
		return fmt.Errorf("update user belt: %w", err)
	}

	err := uc.user.AddBeltPromotion(ctx, u.ID, belt, time.Now())
	if err != nil && !errors.Is(err, model.ErrDuplicateBeltPromotion) {
		return fmt.Errorf("add belt promotion: %w", err)
	}

	if err := uc.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(in.ChatID, in.MessageID, profileText(u), profileKeyboard())
}
