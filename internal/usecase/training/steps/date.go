package steps

import (
	"context"
	"fmt"
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

	var date time.Time

	switch {
	case in.HasCallback && in.CallbackData == callbackDateToday:
		date = time.Now()

		if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
			return err
		}

	case in.HasCallback && in.CallbackData == callbackDateYesterday:
		date = time.Now().AddDate(0, 0, -1)

		if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
			return err
		}

	case in.HasCallback:
		return s.bot.AnswerCallback(in.CallbackID)

	case in.HasMessage:
		parsed, ok := parseDate(strings.TrimSpace(in.Text))
		if !ok {
			return s.bot.Send(in.ChatID, dateNotParsed)
		}

		date = parsed

	default:
		return nil
	}

	d.Date = &date
	d.Step = model.TrainingDraftStepAwaitingType

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update training draft: %w", err)
	}

	return s.bot.SendWithKeyboard(in.ChatID, typeQuestion, typeKeyboard())
}

func dateKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "Сегодня", Data: callbackDateToday},
			dto.Button{Label: "Вчера", Data: callbackDateYesterday},
		),
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
