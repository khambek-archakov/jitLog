// Package history owns the paginated list of a user's already-logged
// trainings — the training:history:page:{n} entry point, reached both from
// the "📋 Мои тренировки" main-menu button and from a card's "← Назад"
// button (see internal/usecase/training/info). It never creates or mutates
// anything; tapping a row hands off to info via its own training:view:{id}
// callback.
package history

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/onboarding/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/training/info"
)

const pageSize = 5

// callbackHistoryPagePrefix is followed by a 0-based page number.
const callbackHistoryPagePrefix = "training:history:page:"

// callbackMenuBack mirrors internal/usecase/onboarding/steps' own private
// constant — tapping it re-shows the main menu, which only onboarding owns.
const callbackMenuBack = "menu:back"

type UseCase struct {
	bot  sender
	repo trainingRepo
}

func New(bot sender, repo trainingRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

func (uc *UseCase) Handle(ctx context.Context, userID int64, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	page, ok := parsePage(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(in.CallbackID)
	}

	trainings, hasMore, err := uc.repo.ListTrainings(ctx, userID, pageSize, page*pageSize)
	if err != nil {
		return fmt.Errorf("list trainings: %w", err)
	}

	if err := uc.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(in.ChatID, in.MessageID, listText(trainings), listKeyboard(trainings, page, hasMore))
}

func listText(trainings []*model.Training) string {
	if len(trainings) == 0 {
		return "Тренировок пока нет."
	}

	return "📋 Мои тренировки"
}

func listKeyboard(trainings []*model.Training, page int, hasMore bool) dto.Keyboard {
	kb := make(dto.Keyboard, 0, len(trainings)+2)

	for _, t := range trainings {
		kb = append(kb, dto.Row(dto.Button{Label: rowLabel(t), Data: fmt.Sprintf("training:view:%d", t.ID)}))
	}

	if nav := navRow(page, hasMore); len(nav) > 0 {
		kb = append(kb, nav)
	}

	kb = append(kb, dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack}))

	return kb
}

func navRow(page int, hasMore bool) []dto.Button {
	var row []dto.Button

	if page > 0 {
		row = append(row, dto.Button{Label: "‹", Data: pageCallback(page - 1)})
	}

	if hasMore {
		row = append(row, dto.Button{Label: "›", Data: pageCallback(page + 1)})
	}

	return row
}

func pageCallback(page int) string {
	return fmt.Sprintf("%s%d", callbackHistoryPagePrefix, page)
}

func rowLabel(t *model.Training) string {
	return fmt.Sprintf("%s · %s · %d мин", t.Date.Format("02.01.2006"), info.TrainingTypeLabel(t.TrainingType), t.DurationMinutes)
}

func parsePage(data string) (int, bool) {
	if !strings.HasPrefix(data, callbackHistoryPagePrefix) {
		return 0, false
	}

	page, err := strconv.Atoi(strings.TrimPrefix(data, callbackHistoryPagePrefix))
	if err != nil || page < 0 {
		return 0, false
	}

	return page, true
}
