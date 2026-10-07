package steps

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/menu"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/info"
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

// callbackTrainingEditPrefix and callbackTrainingDeletePrefix mirror
// internal/usecase/training/update's and .../delete's own private
// constants — by the time either is tapped the draft (and thus this
// scenario's involvement) is long gone, Router sends them there directly.
const (
	callbackTrainingEditPrefix   = "training:edit:"
	callbackTrainingDeletePrefix = "training:delete:"
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
			if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
				return err
			}

			return s.bot.Send(ctx, in.ChatID, "Напиши длительность в минутах, например: 45")
		}

		minutes, ok := durationFromCallback(in.CallbackData)
		if !ok {
			return s.bot.AnswerCallback(ctx, in.CallbackID)
		}

		return s.finish(ctx, d, in, minutes)
	}

	if !in.HasMessage {
		return nil
	}

	minutes, err := strconv.Atoi(strings.TrimSpace(in.Text))
	if err != nil || minutes <= 0 {
		return s.bot.Send(ctx, in.ChatID, durationNotParsed)
	}

	return s.finish(ctx, d, in, minutes)
}

// finish is the wizard's terminal step — it creates the real training row
// and clears the draft, rather than advancing to another question. Rounds
// and notes are deliberately not asked here (they're the two quick
// follow-up actions the confirmation card below offers instead), keeping
// the create flow to exactly three questions: date, type, duration.
func (s *DurationStep) finish(ctx context.Context, d *model.TrainingDraft, in dto.Input, minutes int) error {
	if d.Date == nil {
		return fmt.Errorf("training draft %d has no date set", d.ID)
	}

	t, err := s.repo.CreateTraining(ctx, d.UserID, *d.Date, d.TrainingType, int32(minutes), nil)
	if err != nil {
		return fmt.Errorf("create training: %w", err)
	}

	if err := s.repo.DeleteDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete training draft: %w", err)
	}

	if in.HasCallback {
		if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}
	}

	if err := s.bot.SendWithKeyboard(ctx, in.ChatID, confirmationText(t), confirmationKeyboard(t.ID)); err != nil {
		return err
	}

	// A nudge towards what's next — without this the user has nothing left
	// on screen to tap after saving.
	return s.bot.SendWithKeyboard(ctx, in.ChatID, menu.Text, menu.Keyboard())
}

func (s *DurationStep) handleBack(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	d.Step = model.TrainingDraftStepAwaitingType

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update training draft: %w", err)
	}

	if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(ctx, in.ChatID, typeQuestion, typeKeyboard())
}

func durationKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "60 мин", Data: callbackDuration60},
			dto.Button{Label: "90 мин", Data: callbackDuration90},
			dto.Button{Label: "120 мин", Data: callbackDuration120},
		),
		dto.Row(dto.Button{Label: "Другое", Data: callbackDurationOther}),
		dto.Row(backButton(), cancelButton()),
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

// confirmationText/confirmationSummary are a compact, single-line summary
// — deliberately not info.Body, which is the full multi-line card shown
// everywhere else (history, view, edit). Right after saving there's
// nothing to editorialize yet (no rounds, no notes), so a dense one-liner
// plus the quick-action buttons below it says more with less.
func confirmationText(t *model.Training) string {
	return "✅ Тренировка сохранена!\n" + confirmationSummary(t)
}

func confirmationSummary(t *model.Training) string {
	return fmt.Sprintf(
		"📅 %s · %s · ⏱ %s",
		info.FormatDateShort(t.Date), info.TrainingTypeLabel(t.TrainingType), info.FormatDuration(t.DurationMinutes),
	)
}

func confirmationKeyboard(trainingID int64) dto.Keyboard {
	editPrefix := fmt.Sprintf("%s%d:", callbackTrainingEditPrefix, trainingID)

	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "➕ Раунды", Data: editPrefix + "rounds"},
			dto.Button{Label: "📝 Заметка", Data: editPrefix + "notes"},
		),
		dto.Row(
			dto.Button{Label: "✏️ Изменить", Data: fmt.Sprintf("%s%d", callbackTrainingEditPrefix, trainingID)},
			dto.Button{Label: "🗑 Удалить", Data: fmt.Sprintf("%s%d", callbackTrainingDeletePrefix, trainingID)},
		),
	}
}
