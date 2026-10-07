package competition

import (
	"context"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackAdd mirrors the competition list screen's own private constant —
// the "➕ Добавить" button.
const callbackAdd = "competition:add"

// callbackAddFromCatalog mirrors catalog/list's own private constant —
// its own "➕ Добавить" button starts the exact same wizard, just from a
// different screen. create.Begin tells the two apart by CallbackData to
// know which screen Отмена should return to.
const callbackAddFromCatalog = "competition:add:from_catalog"

// addTrigger starts a fresh tournament dialog from either the
// "🏆 Соревнования" screen's own "➕ Добавить" button or catalog's
// "🔎 Найти соревнование" screen's own "➕ Добавить" button.
type addTrigger struct {
	create competitionCreate
}

func (h *addTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || (in.CallbackData != callbackAdd && in.CallbackData != callbackAddFromCatalog) {
		return model.ErrSkip
	}

	return h.create.Begin(ctx, u, in)
}
