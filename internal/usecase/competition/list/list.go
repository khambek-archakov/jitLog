// Package list owns the "🏆 Соревнования" screen — a user's upcoming
// tournaments, soonest first, plus the entry point into competition/create,
// the hop into competition/history for past ones, and (only when at least
// one published tournament exists) the entry point into catalog/list.
// Tapping a row hands off to competition/info via its own
// competition:view:{id} callback.
package list

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	callbackCompetitionList  = "competition:list"
	callbackCompetitionAdd   = "competition:add"
	callbackHistoryFirstPage = "competition:history:page:0"
	// callbackCatalogList mirrors internal/usecase/catalog/list's own
	// private constant — this screen is the catalog's only entry point.
	callbackCatalogList = "catalog:list"
)

// callbackMenuBack mirrors other scenarios' own private constant.
const callbackMenuBack = "menu:back"

const emptyText = "Пока нет предстоящих соревнований. Добавь своё, чтобы видеть обратный отсчёт и сохранять результат"

type UseCase struct {
	bot  sender
	repo competitionRepo
}

func New(bot sender, repo competitionRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	if in.CallbackData != callbackCompetitionList {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	today := info.Today(u)

	competitions, err := uc.repo.ListUpcoming(ctx, u.ID, today)
	if err != nil {
		return fmt.Errorf("list upcoming competitions: %w", err)
	}

	showCatalog, err := uc.repo.HasPublishedUpcoming(ctx, today)
	if err != nil {
		return fmt.Errorf("check for published upcoming competitions: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, listText(competitions), listKeyboard(competitions, showCatalog),
	)
}

func listText(competitions []*model.UserCompetition) string {
	if len(competitions) == 0 {
		return emptyText
	}

	return "🏆 Мои соревнования"
}

func listKeyboard(competitions []*model.UserCompetition, showCatalog bool) dto.Keyboard {
	kb := make(dto.Keyboard, 0, len(competitions)+3)

	for _, c := range competitions {
		kb = append(kb, dto.Row(dto.Button{Label: rowLabel(c), Data: fmt.Sprintf("competition:view:%d", c.ID)}))
	}

	addRow := []dto.Button{{Label: "➕ Добавить", Data: callbackCompetitionAdd}}
	if showCatalog {
		addRow = append(addRow, dto.Button{Label: "🔎 Найти", Data: callbackCatalogList})
	}

	kb = append(kb, addRow)

	if len(competitions) > 0 {
		kb = append(kb, dto.Row(dto.Button{Label: "Прошедшие", Data: callbackHistoryFirstPage}))
	}

	kb = append(kb, dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack}))

	return kb
}

func rowLabel(c *model.UserCompetition) string {
	return fmt.Sprintf("📌 %s — %s", info.FormatDateRow(c.Date), c.Title)
}
