// Package history owns the paginated list of a user's past tournaments —
// the competition:history:page:{n} entry point, reached from the
// "🏆 Соревнования" screen's own "Прошедшие" button. It never creates or
// mutates anything; tapping a row hands off to competition/info via its
// own competition:view:{id} callback.
package history

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const pageSize = 5

// callbackPagePrefix is followed by a 0-based page number.
const callbackPagePrefix = "competition:history:page:"

// callbackCompetitionList mirrors internal/usecase/competition/list's own
// private constant — "Прошедшие" is reached from there, so "← Назад" goes
// back there too, not all the way to the main menu.
const callbackCompetitionList = "competition:list"

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

	page, ok := parsePage(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	competitions, hasMore, err := uc.repo.ListPast(ctx, u.ID, info.Today(u), pageSize, page*pageSize)
	if err != nil {
		return fmt.Errorf("list past competitions: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, listText(competitions), listKeyboard(competitions, page, hasMore),
	)
}

func listText(competitions []*model.UserCompetition) string {
	if len(competitions) == 0 {
		return "Прошедших турниров пока нет."
	}

	return "Прошедшие турниры"
}

func listKeyboard(competitions []*model.UserCompetition, page int, hasMore bool) dto.Keyboard {
	kb := make(dto.Keyboard, 0, len(competitions)+2)

	for _, c := range competitions {
		kb = append(kb, dto.Row(dto.Button{Label: rowLabel(c), Data: fmt.Sprintf("competition:view:%d", c.ID)}))
	}

	if nav := navRow(page, hasMore); len(nav) > 0 {
		kb = append(kb, nav)
	}

	kb = append(kb, dto.Row(dto.Button{Label: "← Назад", Data: callbackCompetitionList}))

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
	return fmt.Sprintf("%s%d", callbackPagePrefix, page)
}

func rowLabel(c *model.UserCompetition) string {
	label := fmt.Sprintf("%s — %s", info.FormatDateRow(c.Date), c.Title)

	if c.Result != nil && *c.Result != "" {
		label += " · " + *c.Result
	}

	return label
}

func parsePage(data string) (int, bool) {
	if !strings.HasPrefix(data, callbackPagePrefix) {
		return 0, false
	}

	page, err := strconv.Atoi(strings.TrimPrefix(data, callbackPagePrefix))
	if err != nil || page < 0 {
		return 0, false
	}

	return page, true
}
