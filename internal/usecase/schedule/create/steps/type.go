package steps

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/info"
)

const typeQuestion = "Какой тип тренировки?"

const (
	callbackTrainingTypeGi      = "schedule:draft:type:gi"
	callbackTrainingTypeNoGi    = "schedule:draft:type:no_gi"
	callbackTrainingTypeOpenMat = "schedule:draft:type:open_mat"
)

// callbackScheduleList mirrors internal/usecase/schedule/list's own
// private constant — the confirmation screen's only button goes back there.
const callbackScheduleList = "schedule:list"

type TypeStep struct {
	bot  sender
	repo draftRepo
}

func NewType(bot sender, repo draftRepo) *TypeStep {
	return &TypeStep{bot: bot, repo: repo}
}

func (s *TypeStep) Handle(ctx context.Context, d *model.ScheduleDraft, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	if in.CallbackData == callbackBack {
		return s.handleBack(ctx, d, in)
	}

	trainingType, ok := trainingTypeFromCallback(in.CallbackData)
	if !ok {
		return s.bot.AnswerCallback(ctx, in.CallbackID)
	}

	if d.DayOfWeek == nil {
		return fmt.Errorf("schedule draft %d has no day set", d.ID)
	}
	if d.TimeMinutes == nil {
		return fmt.Errorf("schedule draft %d has no time set", d.ID)
	}

	slot, err := s.repo.CreateSlot(ctx, d.UserID, *d.DayOfWeek, *d.TimeMinutes, trainingType)
	if err != nil {
		return fmt.Errorf("create schedule slot: %w", err)
	}

	if err := s.repo.DeleteDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete schedule draft: %w", err)
	}

	if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(ctx, in.ChatID, confirmationText(slot), confirmationKeyboard())
}

func (s *TypeStep) handleBack(ctx context.Context, d *model.ScheduleDraft, in dto.Input) error {
	d.Step = model.ScheduleDraftStepAwaitingTime

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update schedule draft: %w", err)
	}

	if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(ctx, in.ChatID, timeQuestion, timeKeyboard())
}

func typeKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "🥋 Gi", Data: callbackTrainingTypeGi},
			dto.Button{Label: "🥷 No-Gi", Data: callbackTrainingTypeNoGi},
		),
		dto.Row(dto.Button{Label: "🤼 Open Mat", Data: callbackTrainingTypeOpenMat}),
		dto.Row(backButton(), cancelButton()),
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

func confirmationText(s *model.ScheduleSlot) string {
	return "✅ Добавлено: " + info.Body(s)
}

func confirmationKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "📅 К расписанию", Data: callbackScheduleList}),
	}
}
