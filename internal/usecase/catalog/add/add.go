// Package add owns "➕ В мои" — copying a catalog entry into the tapping
// user's own personal list. The new record is a completely independent
// UserCompetition from that point on (editable, deletable, re-submittable)
// — CompetitionID just remembers where it came from.
package add

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

const callbackAddPrefix = "catalog:add:"

type UseCase struct {
	bot  sender
	repo repo
}

func New(bot sender, r repo) *UseCase {
	return &UseCase{bot: bot, repo: r}
}

func (uc *UseCase) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback {
		return nil
	}

	id, ok := parseID(in.CallbackData, callbackAddPrefix)
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

	added, err := uc.repo.CreateUserCompetitionFromCatalog(ctx, u.ID, c.ID, c.Title, c.Date, c.EndDate, c.City, url)
	if err != nil {
		return fmt.Errorf("create user competition from catalog: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	today := competitioninfo.Today(u)
	text := "✅ Добавлено в твои соревнования\n\n" + competitioninfo.Body(added, today)

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, text, competitioninfo.Keyboard(added, today))
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
