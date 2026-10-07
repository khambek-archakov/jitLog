// Package info owns the single-catalog-entry card: the catalog:view:{id}
// entry point, reached by tapping a row in the catalog list. Its only
// action is "➕ В мои" (see sibling package catalog/add) — a catalog entry
// itself is never edited or deleted from here, only by the admin via
// competition/moderate.
package info

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	competitioninfo "github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const notFoundText = "Соревнование не найдено."

const callbackViewPrefix = "catalog:view:"

type UseCase struct {
	bot  sender
	repo catalogRepo
}

func New(bot sender, repo catalogRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	id, ok := parseID(in.CallbackData, callbackViewPrefix)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	c, err := uc.repo.GetCompetition(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}
	if err != nil {
		return fmt.Errorf("get competition: %w", err)
	}

	url, err := uc.repo.GetPrimarySourceURL(ctx, c.ID)
	if err != nil {
		return fmt.Errorf("get primary competition source: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	today := competitioninfo.Today(u)

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, Card(c, url, today), Keyboard(c))
}

func parseID(data, prefix string) (int64, bool) {
	if !strings.HasPrefix(data, prefix) {
		return 0, false
	}

	id, err := strconv.ParseInt(strings.TrimPrefix(data, prefix), 10, 64)
	if err != nil {
		return 0, false
	}

	return id, true
}
