// Package update owns editing a single field of an existing schedule
// slot: the schedule:edit:{id} entry point (menu + per-field pickers, all
// driven by callback data alone) and Continue, which only exists for
// time's "Другое" fallback — the one field whose new value can't ride a
// callback button — and needs the pending model.ScheduleEditDraft Router
// resolved for the current user.
package update

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/info"
)

const notFoundText = "Слот не найден."

const timeNotParsed = "Не смог разобрать время, напиши в формате ЧЧ:ММ, например: 18:30."

// callbackEditPrefix is followed by "{id}", "{id}:day", "{id}:day:{n}",
// "{id}:time", "{id}:time:preset:{token}", "{id}:time:other", "{id}:type"
// or "{id}:type:{value}".
const callbackEditPrefix = "schedule:edit:"

// presetTimes mirrors create/steps' own quick picks, in minutes since
// midnight.
var presetTimes = [...]int16{11 * 60, 12 * 60, 19 * 60, 19*60 + 30, 21 * 60}

type UseCase struct {
	bot  sender
	repo slotRepo
}

func New(bot sender, repo slotRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

// Handle dispatches every schedule:edit:* callback — Router only reaches
// this usecase for that prefix.
func (uc *UseCase) Handle(ctx context.Context, userID int64, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	rest := strings.TrimPrefix(in.CallbackData, callbackEditPrefix)

	idStr, action, _ := strings.Cut(rest, ":")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	s, err := uc.repo.GetSlot(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}
	if err != nil {
		return fmt.Errorf("get schedule slot: %w", err)
	}

	if s.UserID != userID {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}

	switch {
	case action == "":
		return uc.showMenu(ctx, in, s.ID)

	case action == "day" || strings.HasPrefix(action, "day:"):
		return uc.handleDay(ctx, in, s, strings.TrimPrefix(action, "day"))

	case action == "time" || strings.HasPrefix(action, "time:"):
		return uc.handleTime(ctx, in, s, strings.TrimPrefix(action, "time"))

	case action == "type" || strings.HasPrefix(action, "type:"):
		return uc.handleType(ctx, in, s, strings.TrimPrefix(action, "type"))

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

// Continue applies a free-text reply against the pending edit draft — the
// only path where the new time didn't come from a callback button.
func (uc *UseCase) Continue(ctx context.Context, d *model.ScheduleEditDraft, in dto.Input) error {
	if !in.HasMessage {
		return nil
	}

	s, err := uc.repo.GetSlot(ctx, d.SlotID)
	if errors.Is(err, model.ErrNotFound) {
		return uc.repo.DeleteEditDraft(ctx, d.UserID)
	}
	if err != nil {
		return fmt.Errorf("get schedule slot: %w", err)
	}

	minutes, ok := parseTimeText(strings.TrimSpace(in.Text))
	if !ok {
		return uc.bot.Send(ctx, in.ChatID, timeNotParsed)
	}

	return uc.finishEdit(ctx, in, d, s.DayOfWeek, minutes, s.TrainingType)
}

func (uc *UseCase) showMenu(ctx context.Context, in dto.Input, id int64) error {
	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "✏️ Что изменить?", editMenuKeyboard(id))
}

func (uc *UseCase) handleDay(ctx context.Context, in dto.Input, s *model.ScheduleSlot, sub string) error {
	if sub == "" {
		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "📅 Какой день недели?", dayKeyboard(s.ID))
	}

	day, ok := dayFromToken(strings.TrimPrefix(sub, ":"))
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	return uc.applyUpdate(ctx, in, s, day, s.TimeMinutes, s.TrainingType)
}

func (uc *UseCase) handleTime(ctx context.Context, in dto.Input, s *model.ScheduleSlot, sub string) error {
	switch {
	case sub == "":
		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "🕐 Во сколько?", timeKeyboard(s.ID))

	case sub == ":other":
		if err := uc.repo.SetEditDraft(ctx, s.UserID, s.ID); err != nil {
			return fmt.Errorf("set schedule edit draft: %w", err)
		}

		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.Send(ctx, in.ChatID, "Напиши время в формате ЧЧ:ММ, например: 18:30")

	case strings.HasPrefix(sub, ":preset:"):
		minutes, ok := parseTimeToken(strings.TrimPrefix(sub, ":preset:"))
		if !ok {
			return uc.bot.AnswerCallback(ctx, in.CallbackID)
		}

		return uc.applyUpdate(ctx, in, s, s.DayOfWeek, minutes, s.TrainingType)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

func (uc *UseCase) handleType(ctx context.Context, in dto.Input, s *model.ScheduleSlot, sub string) error {
	if sub == "" {
		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "Какой тип тренировки?", typeKeyboard(s.ID))
	}

	newType, ok := trainingTypeFromToken(strings.TrimPrefix(sub, ":"))
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	return uc.applyUpdate(ctx, in, s, s.DayOfWeek, s.TimeMinutes, newType)
}

func (uc *UseCase) applyUpdate(
	ctx context.Context, in dto.Input, s *model.ScheduleSlot, day, minutes int16, trainingType model.TrainingType,
) error {
	updated, err := uc.repo.UpdateSlot(ctx, s.ID, day, minutes, trainingType)
	if err != nil {
		return fmt.Errorf("update schedule slot: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, info.Card(updated), info.Keyboard(updated.ID))
}

func (uc *UseCase) finishEdit(
	ctx context.Context, in dto.Input, d *model.ScheduleEditDraft, day, minutes int16, trainingType model.TrainingType,
) error {
	updated, err := uc.repo.UpdateSlot(ctx, d.SlotID, day, minutes, trainingType)
	if err != nil {
		return fmt.Errorf("update schedule slot: %w", err)
	}

	if err := uc.repo.DeleteEditDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete schedule edit draft: %w", err)
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, info.Card(updated), info.Keyboard(updated.ID))
}

func editMenuKeyboard(id int64) dto.Keyboard {
	prefix := fmt.Sprintf("%s%d:", callbackEditPrefix, id)

	return dto.Keyboard{
		dto.Row(dto.Button{Label: "📅 День недели", Data: prefix + "day"}),
		dto.Row(dto.Button{Label: "🕐 Время", Data: prefix + "time"}),
		dto.Row(dto.Button{Label: "🥋 Тип тренировки", Data: prefix + "type"}),
		dto.Row(dto.Button{Label: "← Назад", Data: fmt.Sprintf("schedule:view:%d", id)}),
	}
}

func dayKeyboard(id int64) dto.Keyboard {
	prefix := fmt.Sprintf("%s%d:day:", callbackEditPrefix, id)

	buttons := make([]dto.Button, 0, 7)
	for day := int16(1); day <= 7; day++ {
		buttons = append(buttons, dto.Button{Label: info.DayLabel(day), Data: fmt.Sprintf("%s%d", prefix, day)})
	}

	return dto.Keyboard{
		buttons,
		dto.Row(dto.Button{Label: "← Назад", Data: fmt.Sprintf("%s%d", callbackEditPrefix, id)}),
	}
}

func timeKeyboard(id int64) dto.Keyboard {
	prefix := fmt.Sprintf("%s%d:time:", callbackEditPrefix, id)

	return dto.Keyboard{
		dto.Row(presetButton(prefix, presetTimes[0]), presetButton(prefix, presetTimes[1]), presetButton(prefix, presetTimes[2])),
		dto.Row(presetButton(prefix, presetTimes[3]), presetButton(prefix, presetTimes[4])),
		dto.Row(dto.Button{Label: "Другое", Data: prefix + "other"}),
		dto.Row(dto.Button{Label: "← Назад", Data: fmt.Sprintf("%s%d", callbackEditPrefix, id)}),
	}
}

func presetButton(prefix string, minutes int16) dto.Button {
	return dto.Button{Label: info.FormatTime(minutes), Data: prefix + "preset:" + formatTimeToken(minutes)}
}

func typeKeyboard(id int64) dto.Keyboard {
	prefix := fmt.Sprintf("%s%d:type:", callbackEditPrefix, id)

	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "🥋 Gi", Data: prefix + "gi"},
			dto.Button{Label: "🥷 No-Gi", Data: prefix + "no_gi"},
		),
		dto.Row(dto.Button{Label: "🤼 Open Mat", Data: prefix + "open_mat"}),
		dto.Row(dto.Button{Label: "← Назад", Data: fmt.Sprintf("%s%d", callbackEditPrefix, id)}),
	}
}

func dayFromToken(token string) (int16, bool) {
	n, err := strconv.Atoi(token)
	if err != nil || n < 1 || n > 7 {
		return 0, false
	}

	return int16(n), true
}

func trainingTypeFromToken(token string) (model.TrainingType, bool) {
	switch token {
	case "gi":
		return model.TrainingTypeGi, true
	case "no_gi":
		return model.TrainingTypeNoGi, true
	case "open_mat":
		return model.TrainingTypeOpenMat, true
	default:
		return model.TrainingTypeNone, false
	}
}

func formatTimeToken(minutes int16) string {
	return fmt.Sprintf("%02d%02d", minutes/60, minutes%60)
}

func parseTimeToken(s string) (int16, bool) {
	if len(s) != 4 {
		return 0, false
	}

	h, err1 := strconv.Atoi(s[:2])
	m, err2 := strconv.Atoi(s[2:])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}

	return int16(h*60 + m), true
}

func parseTimeText(s string) (int16, bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, false
	}

	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}

	return int16(h*60 + m), true
}
