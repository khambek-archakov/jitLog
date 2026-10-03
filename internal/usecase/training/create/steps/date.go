package steps

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/internal/calendar"
)

const (
	dateQuestion  = "Когда была тренировка?"
	dateNotParsed = "Не смог разобрать дату, напиши в формате ДД.ММ или ДД.ММ.ГГГГ."
)

// callbackMenuAddTraining mirrors the main menu's own button data (see
// internal/usecase/onboarding/steps and internal/usecase/router) — it's the
// trigger that kicks this flow off, so DateStep (the first step) needs to
// recognize it too.
const callbackMenuAddTraining = "menu:add_training"

const (
	callbackDateToday     = "training:date:today"
	callbackDateYesterday = "training:date:yesterday"
	callbackDateOther     = "training:date:other"
	callbackDateCancel    = "training:date:cancel"
	callbackDateNoop      = "training:date:noop"
	// callbackDateCalendarPrefix is followed by a "2006-01" month; callbackDatePickPrefix by a "2006-01-02" date.
	callbackDateCalendarPrefix = "training:date:cal:"
	callbackDatePickPrefix     = "training:date:pick:"
)

var dateCalendarCallbacks = calendar.Callbacks{
	MonthPrefix: callbackDateCalendarPrefix,
	DayPrefix:   callbackDatePickPrefix,
	Noop:        callbackDateNoop,
	Cancel:      callbackDateCancel,
}

type DateStep struct {
	bot  sender
	repo draftRepo
}

func NewDate(bot sender, repo draftRepo) *DateStep {
	return &DateStep{bot: bot, repo: repo}
}

func (s *DateStep) Handle(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	if in.HasCallback && in.CallbackData == callbackMenuAddTraining {
		if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return s.bot.SendWithKeyboard(ctx, in.ChatID, dateQuestion, dateKeyboard())
	}

	if in.HasCallback {
		return s.handleCallback(ctx, d, in)
	}

	if !in.HasMessage {
		return nil
	}

	parsed, ok := parseDate(strings.TrimSpace(in.Text))
	if !ok {
		return s.bot.Send(ctx, in.ChatID, dateNotParsed)
	}

	return s.pickDate(ctx, d, in, parsed)
}

func (s *DateStep) handleCallback(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	switch {
	case in.CallbackData == callbackDateNoop:
		return s.bot.AnswerCallback(ctx, in.CallbackID)

	case in.CallbackData == callbackDateToday:
		return s.pickDate(ctx, d, in, time.Now())

	case in.CallbackData == callbackDateYesterday:
		return s.pickDate(ctx, d, in, time.Now().AddDate(0, 0, -1))

	case in.CallbackData == callbackDateOther:
		now := time.Now()
		return s.showCalendar(ctx, in, now.Year(), now.Month())

	case in.CallbackData == callbackDateCancel:
		if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return s.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, dateQuestion, dateKeyboard())

	case strings.HasPrefix(in.CallbackData, callbackDateCalendarPrefix):
		year, month, ok := calendar.ParseYearMonth(strings.TrimPrefix(in.CallbackData, callbackDateCalendarPrefix))
		if !ok {
			return s.bot.AnswerCallback(ctx, in.CallbackID)
		}

		return s.showCalendar(ctx, in, year, month)

	case strings.HasPrefix(in.CallbackData, callbackDatePickPrefix):
		date, ok := calendar.ParseDate(strings.TrimPrefix(in.CallbackData, callbackDatePickPrefix))
		if !ok {
			return s.bot.AnswerCallback(ctx, in.CallbackID)
		}

		return s.pickDate(ctx, d, in, date)

	default:
		return s.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

// showCalendar replaces the in-flight message's keyboard with a calendar for
// year/month, in place — used both for the first "📅 Другая дата" tap and
// for every ‹/› navigation afterwards, so paging months never spams the chat
// with new messages.
func (s *DateStep) showCalendar(ctx context.Context, in dto.Input, year int, month time.Month) error {
	if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return s.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, calendar.Text(year, month), calendar.Keyboard(year, month, dateCalendarCallbacks),
	)
}

func (s *DateStep) pickDate(ctx context.Context, d *model.TrainingDraft, in dto.Input, date time.Time) error {
	d.Date = &date
	d.Step = model.TrainingDraftStepAwaitingType

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update training draft: %w", err)
	}

	if in.HasCallback {
		if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}
	}

	return s.bot.SendWithKeyboard(ctx, in.ChatID, typeQuestion, typeKeyboard())
}

func dateKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "Сегодня", Data: callbackDateToday},
			dto.Button{Label: "Вчера", Data: callbackDateYesterday},
		),
		dto.Row(dto.Button{Label: "📅 Другая дата", Data: callbackDateOther}),
		dto.Row(cancelButton()),
	}
}

func parseDate(text string) (time.Time, bool) {
	if t, err := time.Parse("02.01.2006", text); err == nil {
		return t, true
	}

	t, err := time.Parse("02.01", text)
	if err != nil {
		return time.Time{}, false
	}

	now := time.Now()

	t = t.AddDate(now.Year(), 0, 0)
	if t.After(now) {
		t = t.AddDate(-1, 0, 0)
	}

	return t, true
}
