package steps

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
)

const typeQuestion = "Какой тип тренировки?"

const (
	callbackTrainingTypeGi      = "training:type:gi"
	callbackTrainingTypeNoGi    = "training:type:no_gi"
	callbackTrainingTypeOpenMat = "training:type:open_mat"
)

type TypeStep struct {
	bot  sender
	repo draftRepo
}

func NewType(bot sender, repo draftRepo) *TypeStep {
	return &TypeStep{bot: bot, repo: repo}
}

func (s *TypeStep) Handle(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	if in.CallbackData == callbackBack {
		return s.handleBack(ctx, d, in)
	}

	trainingType, ok := trainingTypeFromCallback(in.CallbackData)
	if !ok {
		return s.bot.AnswerCallback(in.CallbackID)
	}

	d.TrainingType = trainingType
	d.Step = model.TrainingDraftStepAwaitingDuration

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update training draft: %w", err)
	}

	if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(in.ChatID, durationQuestion, durationKeyboard())
}

func (s *TypeStep) handleBack(ctx context.Context, d *model.TrainingDraft, in dto.Input) error {
	d.Step = model.TrainingDraftStepAwaitingDate

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update training draft: %w", err)
	}

	if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(in.ChatID, dateQuestion, dateKeyboard())
}

func typeKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "🥋 Gi", Data: callbackTrainingTypeGi},
			dto.Button{Label: "🥷 No-Gi", Data: callbackTrainingTypeNoGi},
		),
		dto.Row(
			dto.Button{Label: "🤼 Open Mat", Data: callbackTrainingTypeOpenMat},
		),
		dto.Row(backButton()),
	}
}

func trainingTypeFromCallback(data string) (model.TrainingType, bool) {
	switch data {
	case callbackTrainingTypeGi:
		return model.TrainingTypeGi, true
	case callbackTrainingTypeNoGi:
		return model.TrainingTypeNoGi, true
	case callbackTrainingTypeOpenMat:
		return model.TrainingTypeOpenMat, true
	default:
		return model.TrainingTypeNone, false
	}
}
