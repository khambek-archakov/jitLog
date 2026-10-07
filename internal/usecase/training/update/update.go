// Package update owns editing a single field of an already-logged
// training: the training:edit:{id} entry point (menu + per-field pickers,
// all driven by callback data alone) and Continue, which only exists for
// the fields whose new value can't ride a callback button — duration's
// and rounds' own "Другое", and notes — and needs the pending
// model.TrainingEditDraft Router resolved for the current user.
package update

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/internal/calendar"
)

const notFoundText = "Тренировка не найдена."

const roundsNotParsed = "Не смог разобрать число, напиши количество раундов цифрами (0–49)."

// callbackEditPrefix is followed by "{id}", "{id}:date", "{id}:date:cal:…",
// "{id}:date:pick:…", "{id}:type", "{id}:type:{value}", "{id}:duration",
// "{id}:duration:{value|other}", "{id}:rounds", "{id}:rounds:{value|other}",
// "{id}:notes" or "{id}:cancel".
const callbackEditPrefix = "training:edit:"

// callbackEditNoop is the calendar's blank-cell callback — it never parses
// as a valid {id}, so Handle's own id-parsing failure already treats it as
// a harmless no-op; nothing further needs to recognize it explicitly.
const callbackEditNoop = "training:edit:noop"

type UseCase struct {
	bot  sender
	repo trainingRepo
}

func New(bot sender, repo trainingRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

// Handle dispatches every training:edit:* callback — Router only reaches
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

	t, err := uc.repo.GetTraining(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}
	if err != nil {
		return fmt.Errorf("get training: %w", err)
	}

	if t.UserID != userID {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}

	switch {
	case action == "":
		return uc.showMenu(ctx, in, t.ID)

	case action == "date" || strings.HasPrefix(action, "date:"):
		return uc.handleDate(ctx, in, t, strings.TrimPrefix(action, "date"))

	case action == "type" || strings.HasPrefix(action, "type:"):
		return uc.handleType(ctx, in, t, strings.TrimPrefix(action, "type"))

	case action == "duration" || strings.HasPrefix(action, "duration:"):
		return uc.handleDuration(ctx, in, t, strings.TrimPrefix(action, "duration"))

	case action == "rounds" || strings.HasPrefix(action, "rounds:"):
		return uc.handleRounds(ctx, in, t, strings.TrimPrefix(action, "rounds"))

	case action == "notes":
		return uc.promptNotes(ctx, in, t)

	case action == "cancel":
		return uc.cancelEdit(ctx, in, t)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

// Continue applies a free-text reply against the pending edit draft — the
// only path where the new value didn't come from a callback button.
func (uc *UseCase) Continue(ctx context.Context, d *model.TrainingEditDraft, in dto.Input) error {
	if !in.HasMessage {
		return nil
	}

	t, err := uc.repo.GetTraining(ctx, d.TrainingID)
	if errors.Is(err, model.ErrNotFound) {
		return uc.repo.DeleteEditDraft(ctx, d.UserID)
	}
	if err != nil {
		return fmt.Errorf("get training: %w", err)
	}

	switch d.Field {
	case model.TrainingEditFieldDuration:
		minutes, err := strconv.Atoi(strings.TrimSpace(in.Text))
		if err != nil || minutes <= 0 {
			return uc.bot.Send(ctx, in.ChatID, "Не смог разобрать число, напиши длительность в минутах цифрами.")
		}

		return uc.finishEdit(ctx, in, d, t.Date, t.TrainingType, int32(minutes), t.Rounds, t.Notes)

	case model.TrainingEditFieldRounds:
		rounds, ok := parseRounds(in.Text)
		if !ok {
			return uc.bot.Send(ctx, in.ChatID, roundsNotParsed)
		}

		return uc.finishEdit(ctx, in, d, t.Date, t.TrainingType, t.DurationMinutes, &rounds, t.Notes)

	case model.TrainingEditFieldNotes:
		text := strings.TrimSpace(in.Text)

		return uc.finishEdit(ctx, in, d, t.Date, t.TrainingType, t.DurationMinutes, t.Rounds, &text)

	default:
		return uc.repo.DeleteEditDraft(ctx, d.UserID)
	}
}

func (uc *UseCase) showMenu(ctx context.Context, in dto.Input, id int64) error {
	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "✏️ Что изменить?", editMenuKeyboard(id))
}

func (uc *UseCase) handleDate(ctx context.Context, in dto.Input, t *model.Training, sub string) error {
	switch {
	case sub == "":
		now := time.Now()
		return uc.showCalendar(ctx, in, t.ID, now.Year(), now.Month())

	case strings.HasPrefix(sub, ":cal:"):
		year, month, ok := calendar.ParseYearMonth(strings.TrimPrefix(sub, ":cal:"))
		if !ok {
			return uc.bot.AnswerCallback(ctx, in.CallbackID)
		}

		return uc.showCalendar(ctx, in, t.ID, year, month)

	case strings.HasPrefix(sub, ":pick:"):
		date, ok := calendar.ParseDate(strings.TrimPrefix(sub, ":pick:"))
		if !ok {
			return uc.bot.AnswerCallback(ctx, in.CallbackID)
		}

		return uc.applyUpdate(ctx, in, t, date, t.TrainingType, t.DurationMinutes, t.Rounds, t.Notes)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

func (uc *UseCase) showCalendar(ctx context.Context, in dto.Input, id int64, year int, month time.Month) error {
	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, calendar.Text(year, month), calendar.Keyboard(year, month, dateCalendarCallbacks(id)),
	)
}

func (uc *UseCase) handleType(ctx context.Context, in dto.Input, t *model.Training, sub string) error {
	if sub == "" {
		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "Какой тип тренировки?", typeKeyboard(t.ID))
	}

	newType, ok := trainingTypeFromToken(strings.TrimPrefix(sub, ":"))
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	return uc.applyUpdate(ctx, in, t, t.Date, newType, t.DurationMinutes, t.Rounds, t.Notes)
}

func (uc *UseCase) handleDuration(ctx context.Context, in dto.Input, t *model.Training, sub string) error {
	if sub == "" {
		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "⏱ Сколько длилась тренировка?", durationKeyboard(t.ID))
	}

	value := strings.TrimPrefix(sub, ":")

	if value == "other" {
		if err := uc.repo.SetEditDraft(ctx, t.UserID, t.ID, model.TrainingEditFieldDuration); err != nil {
			return fmt.Errorf("set training edit draft: %w", err)
		}

		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.SendWithKeyboard(ctx, in.ChatID, "Напиши длительность в минутах, например: 45", cancelKeyboard(t.ID))
	}

	minutes, err := strconv.Atoi(value)
	if err != nil {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	return uc.applyUpdate(ctx, in, t, t.Date, t.TrainingType, int32(minutes), t.Rounds, t.Notes)
}

// handleRounds mirrors handleDuration — quick picks ride a callback,
// "Другое" falls back to a free-text edit draft.
func (uc *UseCase) handleRounds(ctx context.Context, in dto.Input, t *model.Training, sub string) error {
	if sub == "" {
		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "Сколько было раундов?", roundsKeyboard(t.ID))
	}

	value := strings.TrimPrefix(sub, ":")

	if value == "other" {
		if err := uc.repo.SetEditDraft(ctx, t.UserID, t.ID, model.TrainingEditFieldRounds); err != nil {
			return fmt.Errorf("set training edit draft: %w", err)
		}

		if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return uc.bot.SendWithKeyboard(ctx, in.ChatID, "Напиши количество раундов цифрами, например: 6", cancelKeyboard(t.ID))
	}

	rounds, ok := roundsFromToken(value)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	return uc.applyUpdate(ctx, in, t, t.Date, t.TrainingType, t.DurationMinutes, &rounds, t.Notes)
}

func (uc *UseCase) promptNotes(ctx context.Context, in dto.Input, t *model.Training) error {
	if err := uc.repo.SetEditDraft(ctx, t.UserID, t.ID, model.TrainingEditFieldNotes); err != nil {
		return fmt.Errorf("set training edit draft: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, "Напиши новую заметку:", cancelKeyboard(t.ID))
}

// cancelEdit backs out of whichever free-text prompt is pending (duration,
// rounds or notes — training has only one edit draft per user, so clearing
// it is unambiguous) without requiring any text at all, re-showing the
// training card right in place of the prompt.
func (uc *UseCase) cancelEdit(ctx context.Context, in dto.Input, t *model.Training) error {
	if err := uc.repo.DeleteEditDraft(ctx, t.UserID); err != nil {
		return fmt.Errorf("delete training edit draft: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, info.Card(t), info.Keyboard(t.ID))
}

func cancelKeyboard(id int64) dto.Keyboard {
	return dto.Keyboard{dto.Row(dto.Button{Label: "❌ Отмена", Data: fmt.Sprintf("%s%d:cancel", callbackEditPrefix, id)})}
}

func (uc *UseCase) applyUpdate(
	ctx context.Context, in dto.Input, t *model.Training,
	date time.Time, trainingType model.TrainingType, duration int32, rounds *int16, notes *string,
) error {
	updated, err := uc.repo.UpdateTraining(ctx, t.ID, date, trainingType, duration, rounds, notes)
	if err != nil {
		return fmt.Errorf("update training: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, info.Card(updated), info.Keyboard(updated.ID))
}

func (uc *UseCase) finishEdit(
	ctx context.Context, in dto.Input, d *model.TrainingEditDraft,
	date time.Time, trainingType model.TrainingType, duration int32, rounds *int16, notes *string,
) error {
	updated, err := uc.repo.UpdateTraining(ctx, d.TrainingID, date, trainingType, duration, rounds, notes)
	if err != nil {
		return fmt.Errorf("update training: %w", err)
	}

	if err := uc.repo.DeleteEditDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete training edit draft: %w", err)
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, info.Card(updated), info.Keyboard(updated.ID))
}

func editMenuKeyboard(id int64) dto.Keyboard {
	prefix := fmt.Sprintf("%s%d:", callbackEditPrefix, id)

	return dto.Keyboard{
		dto.Row(dto.Button{Label: "📅 Дата", Data: prefix + "date"}),
		dto.Row(dto.Button{Label: "🥋 Тип тренировки", Data: prefix + "type"}),
		dto.Row(dto.Button{Label: "⏱ Длительность", Data: prefix + "duration"}),
		dto.Row(dto.Button{Label: "🔄 Раунды", Data: prefix + "rounds"}),
		dto.Row(dto.Button{Label: "📝 Заметка", Data: prefix + "notes"}),
		dto.Row(dto.Button{Label: "← Назад", Data: fmt.Sprintf("training:view:%d", id)}),
	}
}

func dateCalendarCallbacks(id int64) calendar.Callbacks {
	base := fmt.Sprintf("%s%d:date:", callbackEditPrefix, id)

	return calendar.Callbacks{
		MonthPrefix: base + "cal:",
		DayPrefix:   base + "pick:",
		Noop:        callbackEditNoop,
		// Отмена just returns to the card — info owns that screen already.
		Cancel: fmt.Sprintf("training:view:%d", id),
	}
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

func durationKeyboard(id int64) dto.Keyboard {
	prefix := fmt.Sprintf("%s%d:duration:", callbackEditPrefix, id)

	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "60 мин", Data: prefix + "60"},
			dto.Button{Label: "90 мин", Data: prefix + "90"},
			dto.Button{Label: "120 мин", Data: prefix + "120"},
		),
		dto.Row(dto.Button{Label: "Другое", Data: prefix + "other"}),
		dto.Row(dto.Button{Label: "← Назад", Data: fmt.Sprintf("%s%d", callbackEditPrefix, id)}),
	}
}

func roundsKeyboard(id int64) dto.Keyboard {
	prefix := fmt.Sprintf("%s%d:rounds:", callbackEditPrefix, id)

	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "3", Data: prefix + "3"},
			dto.Button{Label: "5", Data: prefix + "5"},
			dto.Button{Label: "7", Data: prefix + "7"},
			dto.Button{Label: "10", Data: prefix + "10"},
		),
		dto.Row(dto.Button{Label: "Другое", Data: prefix + "other"}),
		dto.Row(dto.Button{Label: "← Назад", Data: fmt.Sprintf("%s%d", callbackEditPrefix, id)}),
	}
}

func roundsFromToken(token string) (int16, bool) {
	switch token {
	case "3":
		return 3, true
	case "5":
		return 5, true
	case "7":
		return 7, true
	case "10":
		return 10, true
	default:
		return 0, false
	}
}

// parseRounds accepts a plain non-negative integer under 50 — 0 is valid
// (a day with no sparring at all), anything 50+ is almost certainly a typo.
func parseRounds(text string) (int16, bool) {
	n, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || n < 0 || n >= 50 {
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
