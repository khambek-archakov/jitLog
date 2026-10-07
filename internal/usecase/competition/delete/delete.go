// Package delete owns removing a tournament entry: the
// competition:delete:{id} entry point (a confirm screen) and
// competition:delete:confirm:{id} (the actual delete). Both are fully
// stateless — the id rides the callback data the whole way, same as
// training/delete and schedule/delete.
package delete

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const notFoundText = "Соревнование не найдено."

const (
	callbackDeletePrefix        = "competition:delete:"
	callbackDeleteConfirmPrefix = "competition:delete:confirm:"
	// callbackCompetitionList mirrors internal/usecase/competition/list's
	// own private constant — after a successful delete there's nothing
	// left to show a card for.
	callbackCompetitionList = "competition:list"
)

type UseCase struct {
	bot  sender
	repo competitionRepo
}

func New(bot sender, repo competitionRepo) *UseCase {
	return &UseCase{bot: bot, repo: repo}
}

// Handle dispatches every competition:delete:* callback — Router only
// reaches this usecase for that prefix.
func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	if strings.HasPrefix(in.CallbackData, callbackDeleteConfirmPrefix) {
		return uc.confirm(ctx, u, in)
	}

	id, ok := parseID(in.CallbackData, callbackDeletePrefix)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	c, err := uc.getOwnCompetition(ctx, id, u.ID)
	if err != nil {
		return err
	}
	if c == nil {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, confirmText(c, info.Today(u)), confirmKeyboard(c.ID),
	)
}

func (uc *UseCase) confirm(ctx context.Context, u *model.User, in dto.Input) error {
	id, ok := parseID(in.CallbackData, callbackDeleteConfirmPrefix)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	c, err := uc.getOwnCompetition(ctx, id, u.ID)
	if err != nil {
		return err
	}
	if c == nil {
		return uc.bot.AnswerCallbackWithText(ctx, in.CallbackID, notFoundText)
	}

	if err := uc.repo.DeleteUserCompetition(ctx, id); err != nil {
		return fmt.Errorf("delete user competition: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(ctx, in.ChatID, in.MessageID, "🗑 Соревнование удалено.", deletedKeyboard())
}

// getOwnCompetition fetches a competition and checks it belongs to userID,
// returning (nil, nil) for "not found or not yours" so callers can send the
// same notFoundText either way without leaking which case it was.
func (uc *UseCase) getOwnCompetition(ctx context.Context, id, userID int64) (*model.UserCompetition, error) {
	c, err := uc.repo.GetUserCompetition(ctx, id)
	if errors.Is(err, model.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user competition: %w", err)
	}

	if c.UserID != userID {
		return nil, nil
	}

	return c, nil
}

func confirmText(c *model.UserCompetition, today time.Time) string {
	return "🗑 Удалить соревнование?\n\n" + info.Body(c, today)
}

func confirmKeyboard(id int64) dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "Да, удалить", Data: fmt.Sprintf("%s%d", callbackDeleteConfirmPrefix, id)}),
		dto.Row(dto.Button{Label: "Отмена", Data: fmt.Sprintf("competition:view:%d", id)}),
	}
}

func deletedKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "← Соревнования", Data: callbackCompetitionList}),
	}
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
