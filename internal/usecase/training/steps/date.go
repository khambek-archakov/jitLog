package steps

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
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

type DateStep struct {
	bot  sender
	repo draftRepo
}

func NewDate(bot sender, repo draftRepo) *DateStep {
	return &DateStep{bot: bot, repo: repo}
}

func (s *DateStep) Handle(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	if in.HasCallback && in.CallbackData == callbackMenuAddTraining {
		if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
			return err
		}

		return s.bot.SendWithKeyboard(in.ChatID, dateQuestion, dateKeyboard())
	}

	if in.HasCallback {
		return s.handleCallback(ctx, d, in)
	}

	if !in.HasMessage {
		return nil
	}

	parsed, ok := parseDate(strings.TrimSpace(in.Text))
	if !ok {
		return s.bot.Send(in.ChatID, dateNotParsed)
	}

	return s.pickDate(ctx, d, in, parsed)
}

func (s *DateStep) handleCallback(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	switch {
	case in.CallbackData == callbackDateNoop:
		return s.bot.AnswerCallback(in.CallbackID)

	case in.CallbackData == callbackDateToday:
		return s.pickDate(ctx, d, in, time.Now())

	case in.CallbackData == callbackDateYesterday:
		return s.pickDate(ctx, d, in, time.Now().AddDate(0, 0, -1))

	case in.CallbackData == callbackDateOther:
		now := time.Now()
		return s.showCalendar(in, now.Year(), now.Month())

	case in.CallbackData == callbackDateCancel:
		if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
			return err
		}

		return s.bot.EditMessageWithKeyboard(in.ChatID, in.MessageID, dateQuestion, dateKeyboard())

	case strings.HasPrefix(in.CallbackData, callbackDateCalendarPrefix):
		year, month, ok := parseYearMonth(strings.TrimPrefix(in.CallbackData, callbackDateCalendarPrefix))
		if !ok {
			return s.bot.AnswerCallback(in.CallbackID)
		}

		return s.showCalendar(in, year, month)

	case strings.HasPrefix(in.CallbackData, callbackDatePickPrefix):
		date, ok := parseDateISO(strings.TrimPrefix(in.CallbackData, callbackDatePickPrefix))
		if !ok {
			return s.bot.AnswerCallback(in.CallbackID)
		}

		return s.pickDate(ctx, d, in, date)

	default:
		return s.bot.AnswerCallback(in.CallbackID)
	}
}

// showCalendar replaces the in-flight message's keyboard with a calendar for
// year/month, in place — used both for the first "📅 Другая дата" tap and
// for every ‹/› navigation afterwards, so paging months never spams the chat
// with new messages.
func (s *DateStep) showCalendar(in dto.Input, year int, month time.Month) error {
	if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return s.bot.EditMessageWithKeyboard(in.ChatID, in.MessageID, calendarText(year, month), calendarKeyboard(year, month))
}

func (s *DateStep) pickDate(ctx context.Context, d *model.TrainingDraft, in dto.Input, date time.Time) error {
	d.Date = &date
	d.Step = model.TrainingDraftStepAwaitingType

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update training draft: %w", err)
	}

	if in.HasCallback {
		if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
			return err
		}
	}

	return s.bot.SendWithKeyboard(in.ChatID, typeQuestion, typeKeyboard())
}

func dateKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "Сегодня", Data: callbackDateToday},
			dto.Button{Label: "Вчера", Data: callbackDateYesterday},
		),
		dto.Row(dto.Button{Label: "📅 Другая дата", Data: callbackDateOther}),
	}
}

var russianMonthsNominative = [...]string{
	"Январь", "Февраль", "Март", "Апрель", "Май", "Июнь",
	"Июль", "Август", "Сентябрь", "Октябрь", "Ноябрь", "Декабрь",
}

func calendarText(year int, month time.Month) string {
	return fmt.Sprintf("📅 %s %d", russianMonthsNominative[month-1], year)
}

// calendarKeyboard lays out a month as: nav row (‹ Month ›), a weekday
// header row, one row per calendar week, and a Cancel row. Days outside the
// month and days after today (see the future-date rule below) are rendered
// as blank, unclickable cells so the grid stays aligned.
func calendarKeyboard(year int, month time.Month) dto.Keyboard {
	keyboard := dto.Keyboard{
		navRow(year, month),
		weekdayRow(),
	}

	keyboard = append(keyboard, dayRows(year, month, time.Now())...)
	keyboard = append(keyboard, dto.Row(dto.Button{Label: "Отмена", Data: callbackDateCancel}))

	return keyboard
}

func navRow(year int, month time.Month) []dto.Button {
	prevYear, prevMonth := addMonths(year, month, -1)

	// No picking a training that hasn't happened yet — ›  is disabled once
	// we're already showing the current month.
	nextData := callbackDateNoop
	if year*12+int(month) < time.Now().Year()*12+int(time.Now().Month()) {
		nextYear, nextMonth := addMonths(year, month, 1)
		nextData = monthCallback(nextYear, nextMonth)
	}

	return dto.Row(
		dto.Button{Label: "‹", Data: monthCallback(prevYear, prevMonth)},
		dto.Button{Label: russianMonthsNominative[month-1], Data: callbackDateNoop},
		dto.Button{Label: "›", Data: nextData},
	)
}

func weekdayRow() []dto.Button {
	return dto.Row(
		dto.Button{Label: "Пн", Data: callbackDateNoop},
		dto.Button{Label: "Вт", Data: callbackDateNoop},
		dto.Button{Label: "Ср", Data: callbackDateNoop},
		dto.Button{Label: "Чт", Data: callbackDateNoop},
		dto.Button{Label: "Пт", Data: callbackDateNoop},
		dto.Button{Label: "Сб", Data: callbackDateNoop},
		dto.Button{Label: "Вс", Data: callbackDateNoop},
	)
}

func dayRows(year int, month time.Month, now time.Time) [][]dto.Button {
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	// time.Weekday: Sunday=0..Saturday=6 — shift so Monday=0..Sunday=6.
	leading := (int(first.Weekday()) + 6) % 7
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	blank := dto.Button{Label: " ", Data: callbackDateNoop}

	var rows [][]dto.Button

	row := make([]dto.Button, 0, 7)
	for range leading {
		row = append(row, blank)
	}

	for day := 1; day <= daysInMonth; day++ {
		date := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)

		btn := blank
		if !date.After(today) {
			btn = dto.Button{Label: strconv.Itoa(day), Data: dayCallback(year, month, day)}
		}

		row = append(row, btn)

		if len(row) == 7 {
			rows = append(rows, row)
			row = make([]dto.Button, 0, 7)
		}
	}

	if len(row) > 0 {
		for len(row) < 7 {
			row = append(row, blank)
		}

		rows = append(rows, row)
	}

	return rows
}

func addMonths(year int, month time.Month, delta int) (int, time.Month) {
	t := time.Date(year, month+time.Month(delta), 1, 0, 0, 0, 0, time.UTC)
	return t.Year(), t.Month()
}

func monthCallback(year int, month time.Month) string {
	return fmt.Sprintf("%s%04d-%02d", callbackDateCalendarPrefix, year, int(month))
}

func dayCallback(year int, month time.Month, day int) string {
	return fmt.Sprintf("%s%04d-%02d-%02d", callbackDatePickPrefix, year, int(month), day)
}

func parseYearMonth(s string) (int, time.Month, bool) {
	t, err := time.Parse("2006-01", s)
	if err != nil {
		return 0, 0, false
	}

	return t.Year(), t.Month(), true
}

func parseDateISO(s string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, false
	}

	return t, true
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
