// Package create owns the "➕ Добавить соревнование" wizard: title → date, then
// an immediate save. Mirrors internal/usecase/training/create's shape
// (step-dispatch-map, centralized cancel handling) but is deliberately
// just two questions — everything else is added after the fact via the
// confirmation card's own quick-action buttons.
package create

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/create/steps"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/menu"
)

// callbackCancel mirrors steps' own private constant — every step shows
// the button, but it's handled here, once.
const callbackCancel = "competition:draft:cancel"

// callbackAddFromCatalog mirrors catalog/list's own private constant —
// its "➕ Добавить" button starts this exact same wizard, just remembering
// to come back there instead of the main menu when cancelled.
const callbackAddFromCatalog = "competition:add:from_catalog"

// callbackCatalogList mirrors catalog/list's own private constant — Отмена
// re-renders that screen when the wizard was started from there.
const callbackCatalogList = "catalog:list"

type UseCase struct {
	bot         sender
	repo        draftRepo
	catalogList catalogList
	handlers    map[model.UserCompetitionDraftStep]handler
}

func New(bot sender, repo draftRepo, catalogList catalogList) *UseCase {
	return &UseCase{
		bot:         bot,
		repo:        repo,
		catalogList: catalogList,
		handlers: map[model.UserCompetitionDraftStep]handler{
			model.UserCompetitionDraftStepAwaitingTitle: steps.NewTitle(bot, repo),
			model.UserCompetitionDraftStepAwaitingDate:  steps.NewDate(bot, repo),
		},
	}
}

func (uc *UseCase) Begin(ctx context.Context, u *model.User, in dto.Input) error {
	fromCatalog := in.HasCallback && in.CallbackData == callbackAddFromCatalog

	d, err := uc.repo.CreateDraft(ctx, u.ID, fromCatalog)
	if err != nil {
		return fmt.Errorf("create user competition draft: %w", err)
	}

	return uc.dispatch(ctx, u, d, in)
}

func (uc *UseCase) Continue(ctx context.Context, u *model.User, d *model.UserCompetitionDraft, in dto.Input) error {
	return uc.dispatch(ctx, u, d, in)
}

func (uc *UseCase) dispatch(ctx context.Context, u *model.User, d *model.UserCompetitionDraft, in dto.Input) error {
	if in.HasCallback && in.CallbackData == callbackCancel {
		return uc.cancel(ctx, u, d, in)
	}

	h, ok := uc.handlers[d.Step]
	if !ok {
		return nil
	}

	return h.Handle(ctx, u, d, in)
}

func (uc *UseCase) cancel(ctx context.Context, u *model.User, d *model.UserCompetitionDraft, in dto.Input) error {
	if err := uc.repo.DeleteDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete user competition draft: %w", err)
	}

	if d.FromCatalog {
		catalogIn := in
		catalogIn.CallbackData = callbackCatalogList

		return uc.catalogList.Handle(ctx, u, catalogIn)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, menu.Text, menu.Keyboard())
}
