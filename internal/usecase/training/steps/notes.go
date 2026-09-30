package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
)

const (
	notesQuestion     = "Есть что добавить? Например, что отрабатывал."
	callbackSkipNotes = "training:notes:skip"
	doneText          = "Записал! Тренировка добавлена 💪"
)

type NotesStep struct {
	bot  sender
	repo draftRepo
}

func NewNotes(bot sender, repo draftRepo) *NotesStep {
	return &NotesStep{bot: bot, repo: repo}
}

func (s *NotesStep) Handle(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	var notes *string

	switch {
	case in.HasCallback && in.CallbackData == callbackSkipNotes:
		if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
			return err
		}

	case in.HasCallback:
		return s.bot.AnswerCallback(in.CallbackID)

	case in.HasMessage:
		text := strings.TrimSpace(in.Text)
		if text != "" {
			notes = &text
		}

	default:
		return nil
	}

	if d.Date == nil {
		return fmt.Errorf("training draft %d has no date set", d.ID)
	}
	if d.DurationMinutes == nil {
		return fmt.Errorf("training draft %d has no duration set", d.ID)
	}

	if _, err := s.repo.CreateTraining(ctx, d.UserID, *d.Date, d.TrainingType, *d.DurationMinutes, notes); err != nil {
		return fmt.Errorf("create training: %w", err)
	}

	if err := s.repo.DeleteDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete training draft: %w", err)
	}

	return s.bot.Send(in.ChatID, doneText)
}

func notesKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "Пропустить", Data: callbackSkipNotes}),
	}
}
