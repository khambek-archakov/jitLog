// Package update owns editing a single field of an existing tournament
// entry: the competition:edit:{id} entry point (a field-picker menu, all
// driven by callback data) and Continue, which handles every field's
// actual new value — unlike training/schedule, *every* field here is free
// text (no quick-pick buttons anywhere), so Continue is reached for all
// six, not just a couple of leaves.
package update

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/urlnorm"
)

const notFoundText = "Турнир не найден."

const (
	titleNotParsed     = "Название должно быть от 1 до 100 символов, напиши ещё раз."
	dateNotParsed      = "Не смог разобрать дату, напиши в формате ДД.ММ или ДД.ММ.ГГГГ."
	endDateBeforeStart = "Дата окончания не может быть раньше даты начала, напиши ещё раз."
	startDateAfterEnd  = "Дата начала не может быть позже даты окончания, напиши ещё раз."
	cityNotParsed      = "Слишком длинно — город можно описать максимум 80 символами."
	urlNotParsed       = "Ссылка должна начинаться с http:// или https://, напиши ещё раз."
	resultNotParsed    = "Слишком длинно — результат можно описать максимум 200 символами."
)

// callbackEditPrefix is followed by "{id}", "{id}:title", "{id}:date",
// "{id}:end_date", "{id}:city", "{id}:url" or "{id}:result".
const callbackEditPrefix = "competition:edit:"

type UseCase struct {
	bot  sender
	repo competitionRepo
}

func New(bot sender, repo competitionRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

// Handle dispatches every competition:edit:* callback — Router only
// reaches this usecase for that prefix.
func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	rest := strings.TrimPrefix(in.CallbackData, callbackEditPrefix)

	idStr, action, _ := strings.Cut(rest, ":")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	c, err := uc.repo.GetUserCompetition(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}
	if err != nil {
		return fmt.Errorf("get user competition: %w", err)
	}

	if c.UserID != u.ID {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}

	switch action {
	case "":
		return uc.showMenu(ctx, in, c.ID)

	case "title":
		return uc.promptField(ctx, in, c, model.UserCompetitionEditFieldTitle, "Как называется турнир?")

	case "date":
		return uc.promptField(ctx, in, c, model.UserCompetitionEditFieldDate, "Когда он проходит? (ДД.ММ или ДД.ММ.ГГГГ)")

	case "end_date":
		return uc.promptField(
			ctx, in, c, model.UserCompetitionEditFieldEndDate, "Когда он заканчивается? (ДД.ММ или ДД.ММ.ГГГГ)",
		)

	case "city":
		return uc.promptField(ctx, in, c, model.UserCompetitionEditFieldCity, "В каком городе?")

	case "url":
		return uc.promptField(ctx, in, c, model.UserCompetitionEditFieldURL, "Пришли ссылку на турнир.")

	case "result":
		return uc.promptField(ctx, in, c, model.UserCompetitionEditFieldResult, "Какой результат?")

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

// Continue applies a free-text reply against whichever field the pending
// edit draft names.
func (uc *UseCase) Continue(ctx context.Context, u *model.User, d *model.UserCompetitionEditDraft, in dto.Input) error {
	if !in.HasMessage {
		return nil
	}

	c, err := uc.repo.GetUserCompetition(ctx, d.UserCompetitionID)
	if errors.Is(err, model.ErrNotFound) {
		return uc.repo.DeleteEditDraft(ctx, d.UserID)
	}
	if err != nil {
		return fmt.Errorf("get user competition: %w", err)
	}

	switch d.Field {
	case model.UserCompetitionEditFieldTitle:
		title, ok := parseTitle(in.Text)
		if !ok {
			return uc.bot.Send(ctx, in.ChatID, titleNotParsed)
		}

		return uc.finishEdit(ctx, u, in, d, title, c.Date, c.EndDate, c.City, c.URL, c.Result)

	case model.UserCompetitionEditFieldDate:
		date, ok := parseDate(strings.TrimSpace(in.Text))
		if !ok {
			return uc.bot.Send(ctx, in.ChatID, dateNotParsed)
		}
		if c.EndDate != nil && date.After(*c.EndDate) {
			return uc.bot.Send(ctx, in.ChatID, startDateAfterEnd)
		}

		return uc.finishEdit(ctx, u, in, d, c.Title, date, c.EndDate, c.City, c.URL, c.Result)

	case model.UserCompetitionEditFieldEndDate:
		endDate, ok := parseDate(strings.TrimSpace(in.Text))
		if !ok {
			return uc.bot.Send(ctx, in.ChatID, dateNotParsed)
		}
		if endDate.Before(c.Date) {
			return uc.bot.Send(ctx, in.ChatID, endDateBeforeStart)
		}

		return uc.finishEdit(ctx, u, in, d, c.Title, c.Date, &endDate, c.City, c.URL, c.Result)

	case model.UserCompetitionEditFieldCity:
		city, ok := parseOptional(in.Text, 80)
		if !ok {
			return uc.bot.Send(ctx, in.ChatID, cityNotParsed)
		}

		return uc.finishEdit(ctx, u, in, d, c.Title, c.Date, c.EndDate, city, c.URL, c.Result)

	case model.UserCompetitionEditFieldURL:
		normalized, ok := urlnorm.Normalize(in.Text)
		if !ok {
			return uc.bot.Send(ctx, in.ChatID, urlNotParsed)
		}

		return uc.finishEdit(ctx, u, in, d, c.Title, c.Date, c.EndDate, c.City, &normalized, c.Result)

	case model.UserCompetitionEditFieldResult:
		result, ok := parseOptional(in.Text, 200)
		if !ok {
			return uc.bot.Send(ctx, in.ChatID, resultNotParsed)
		}

		return uc.finishEdit(ctx, u, in, d, c.Title, c.Date, c.EndDate, c.City, c.URL, result)

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

func (uc *UseCase) promptField(
	ctx context.Context, in dto.Input, c *model.UserCompetition, field model.UserCompetitionEditField, question string,
) error {
	if err := uc.repo.SetEditDraft(ctx, c.UserID, c.ID, field); err != nil {
		return fmt.Errorf("set user competition edit draft: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.Send(ctx, in.ChatID, question)
}

func (uc *UseCase) finishEdit(
	ctx context.Context, u *model.User, in dto.Input, d *model.UserCompetitionEditDraft,
	title string, date time.Time, endDate *time.Time, city, url, result *string,
) error {
	updated, err := uc.repo.UpdateUserCompetition(ctx, d.UserCompetitionID, title, date, endDate, city, url, result)
	if err != nil {
		return fmt.Errorf("update user competition: %w", err)
	}

	if err := uc.repo.DeleteEditDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete user competition edit draft: %w", err)
	}

	today := info.Today(u)

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, info.Card(updated, today), info.Keyboard(updated, today))
}

func editMenuKeyboard(id int64) dto.Keyboard {
	prefix := fmt.Sprintf("%s%d:", callbackEditPrefix, id)

	return dto.Keyboard{
		dto.Row(dto.Button{Label: "Название", Data: prefix + "title"}),
		dto.Row(dto.Button{Label: "Дата", Data: prefix + "date"}),
		dto.Row(dto.Button{Label: "Дата окончания", Data: prefix + "end_date"}),
		dto.Row(dto.Button{Label: "Город", Data: prefix + "city"}),
		dto.Row(dto.Button{Label: "Ссылка", Data: prefix + "url"}),
		dto.Row(dto.Button{Label: "Результат", Data: prefix + "result"}),
		dto.Row(dto.Button{Label: "← Назад", Data: fmt.Sprintf("competition:view:%d", id)}),
	}
}

func parseTitle(text string) (string, bool) {
	title := strings.TrimSpace(text)

	n := utf8.RuneCountInString(title)
	if n < 1 || n > 100 {
		return "", false
	}

	return title, true
}

// parseOptional trims text and accepts anything up to maxRunes — empty
// clears the field (nil), matching how every other optional free-text
// field in this repo treats a blank reply.
func parseOptional(text string, maxRunes int) (*string, bool) {
	trimmed := strings.TrimSpace(text)
	if utf8.RuneCountInString(trimmed) > maxRunes {
		return nil, false
	}

	if trimmed == "" {
		return nil, true
	}

	return &trimmed, true
}

// parseDate accepts "02.01.2006" or bare "02.01" (current year assumed) —
// unlike training's own date parser, it never rolls a bare day.month back
// a year for looking "future", since future tournament dates are the
// common case here, not a typo.
func parseDate(text string) (time.Time, bool) {
	if t, err := time.Parse("02.01.2006", text); err == nil {
		return t, true
	}

	t, err := time.Parse("02.01", text)
	if err != nil {
		return time.Time{}, false
	}

	return t.AddDate(time.Now().Year(), 0, 0), true
}
