package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/info"
)

const (
	notesQuestion     = "📝 Хочешь добавить заметку? Например, что отрабатывал."
	callbackAddNotes  = "training:notes:add"
	callbackSkipNotes = "training:notes:skip"
)

// callbackTrainingEditPrefix and callbackTrainingDeletePrefix mirror
// internal/usecase/training/update's and .../delete's own private
// constants — by the time either is tapped the draft (and thus this
// scenario's involvement) is long gone, Router sends them there directly.
const (
	callbackTrainingEditPrefix   = "training:edit:"
	callbackTrainingDeletePrefix = "training:delete:"
)

type NotesStep struct {
	bot  sender
	repo draftRepo
}

func NewNotes(bot sender, repo draftRepo) *NotesStep {
	return &NotesStep{bot: bot, repo: repo}
}

func (s *NotesStep) Handle(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	if in.HasCallback && in.CallbackData == callbackBack {
		return s.handleBack(ctx, d, in)
	}

	// "Добавить заметку" is just a nudge — it doesn't finish the dialog,
	// the user still types the actual note as a normal message afterwards.
	if in.HasCallback && in.CallbackData == callbackAddNotes {
		if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
			return err
		}

		return s.bot.Send(in.ChatID, "Напиши заметку:")
	}

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

	t, err := s.repo.CreateTraining(ctx, d.UserID, *d.Date, d.TrainingType, *d.DurationMinutes, notes)
	if err != nil {
		return fmt.Errorf("create training: %w", err)
	}

	if err := s.repo.DeleteDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete training draft: %w", err)
	}

	return s.bot.SendWithKeyboard(in.ChatID, confirmationText(t), confirmationKeyboard(t.ID))
}

func (s *NotesStep) handleBack(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	d.Step = model.TrainingDraftStepAwaitingDuration

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update training draft: %w", err)
	}

	if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(in.ChatID, durationQuestion, durationKeyboard())
}

func notesKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "✏️ Добавить заметку", Data: callbackAddNotes}),
		dto.Row(dto.Button{Label: "Пропустить", Data: callbackSkipNotes}),
		dto.Row(backButton()),
	}
}

func confirmationText(t *model.Training) string {
	return "✅ Тренировка сохранена!\n\n" + info.Body(t)
}

func confirmationKeyboard(trainingID int64) dto.Keyboard {
	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "✏️ Изменить", Data: fmt.Sprintf("%s%d", callbackTrainingEditPrefix, trainingID)},
			dto.Button{Label: "🗑 Удалить", Data: fmt.Sprintf("%s%d", callbackTrainingDeletePrefix, trainingID)},
		),
	}
}
