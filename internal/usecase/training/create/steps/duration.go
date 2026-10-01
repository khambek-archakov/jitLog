package steps

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	durationQuestion  = "⏱ Сколько длилась тренировка?"
	durationNotParsed = "Не смог разобрать число, напиши длительность в минутах цифрами."
)

const (
	callbackDuration60    = "training:duration:60"
	callbackDuration90    = "training:duration:90"
	callbackDuration120   = "training:duration:120"
	callbackDurationOther = "training:duration:other"
)

type DurationStep struct {
	bot  sender
	repo draftRepo
}

func NewDuration(bot sender, repo draftRepo) *DurationStep {
	return &DurationStep{bot: bot, repo: repo}
}

func (s *DurationStep) Handle(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	if in.HasCallback {
		if in.CallbackData == callbackBack {
			return s.handleBack(ctx, d, in)
		}

		if in.CallbackData == callbackDurationOther {
			if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
				return err
			}

			return s.bot.Send(in.ChatID, "Напиши длительность в минутах, например: 45")
		}

		minutes, ok := durationFromCallback(in.CallbackData)
		if !ok {
			return s.bot.AnswerCallback(in.CallbackID)
		}

		return s.saveDuration(ctx, d, in, minutes)
	}

	if !in.HasMessage {
		return nil
	}

	minutes, err := strconv.Atoi(strings.TrimSpace(in.Text))
	if err != nil || minutes <= 0 {
		return s.bot.Send(in.ChatID, durationNotParsed)
	}

	return s.saveDuration(ctx, d, in, minutes)
}

func (s *DurationStep) saveDuration(ctx context.Context, d *model.TrainingDraft, in dto.Input, minutes int) error {
	duration := int32(minutes)
	d.DurationMinutes = &duration
	d.Step = model.TrainingDraftStepAwaitingNotes

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update training draft: %w", err)
	}

	if in.HasCallback {
		if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
			return err
		}
	}

	return s.bot.SendWithKeyboard(in.ChatID, notesQuestion, notesKeyboard())
}

func (s *DurationStep) handleBack(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	d.Step = model.TrainingDraftStepAwaitingType

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update training draft: %w", err)
	}

	if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(in.ChatID, typeQuestion, typeKeyboard())
}

func durationKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "60 мин", Data: callbackDuration60},
			dto.Button{Label: "90 мин", Data: callbackDuration90},
			dto.Button{Label: "120 мин", Data: callbackDuration120},
		),
		dto.Row(dto.Button{Label: "Другое", Data: callbackDurationOther}),
		dto.Row(backButton()),
	}
}

func durationFromCallback(data string) (int, bool) {
	switch data {
	case callbackDuration60:
		return 60, true
	case callbackDuration90:
		return 90, true
	case callbackDuration120:
		return 120, true
	default:
		return 0, false
	}
}
