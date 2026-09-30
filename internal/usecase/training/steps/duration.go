package steps

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
)

const (
	durationQuestion  = "Сколько минут длилась тренировка?"
	durationNotParsed = "Не смог разобрать число, напиши длительность в минутах цифрами."
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
		return s.bot.AnswerCallback(in.CallbackID)
	}

	if !in.HasMessage {
		return nil
	}

	minutes, err := strconv.Atoi(strings.TrimSpace(in.Text))
	if err != nil || minutes <= 0 {
		return s.bot.Send(in.ChatID, durationNotParsed)
	}

	duration := int32(minutes)
	d.DurationMinutes = &duration
	d.Step = model.TrainingDraftStepAwaitingNotes

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update training draft: %w", err)
	}

	return s.bot.SendWithKeyboard(in.ChatID, notesQuestion, notesKeyboard())
}
