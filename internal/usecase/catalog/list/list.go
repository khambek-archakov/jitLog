// Package list owns the "🔎 Найти турнир" screen — published, upcoming
// tournaments anyone has submitted, filtered by city. The filter defaults
// to the viewer's own User.City but can be overridden per-view (see
// CatalogViewFilter) without ever touching the profile — changing it here
// is a one-way street away from that default, not into it.
package list

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const pageSize = 5

const (
	callbackList       = "catalog:list"
	callbackPagePrefix = "catalog:list:page:"
	callbackCityPrompt = "catalog:city:prompt"
	callbackCityAll    = "catalog:city:all"
	callbackCityCancel = "catalog:city:cancel"
)

// callbackCompetitionList mirrors internal/usecase/competition/list's own
// private constant — this screen is only ever reached from "🏆 Соревнования",
// so "← Назад" goes back there, not to the main menu.
const callbackCompetitionList = "competition:list"

const cityTooLong = "Слишком длинно 🤔\nГород — максимум 80 символов."

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

	switch {
	case in.CallbackData == callbackList:
		return uc.showPage(ctx, u, in, 0)

	case strings.HasPrefix(in.CallbackData, callbackPagePrefix):
		page, ok := parsePage(in.CallbackData)
		if !ok {
			return uc.bot.AnswerCallback(ctx, in.CallbackID)
		}

		return uc.showPage(ctx, u, in, page)

	case in.CallbackData == callbackCityPrompt:
		return uc.promptCity(ctx, u, in)

	case in.CallbackData == callbackCityAll:
		return uc.setCity(ctx, u, in, nil)

	case in.CallbackData == callbackCityCancel:
		return uc.cancelCity(ctx, u, in)

	default:
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}
}

// Continue applies a free-text city name against the pending catalog city
// draft — the only path where the filter's new value didn't come from a
// callback button. Unlike Handle's showPage, this sends a fresh message
// rather than editing one in place, since a free-text reply isn't an
// editable bot message.
func (uc *UseCase) Continue(ctx context.Context, u *model.User, _ *model.CatalogCityDraft, in dto.Input) error {
	if !in.HasMessage {
		return nil
	}

	city := strings.TrimSpace(in.Text)
	if utf8.RuneCountInString(city) > 80 {
		return uc.bot.Send(ctx, in.ChatID, cityTooLong)
	}
	if city == "" {
		return uc.bot.Send(ctx, in.ChatID, "Напиши название города.")
	}

	if err := uc.repo.SetViewFilter(ctx, u.ID, &city); err != nil {
		return fmt.Errorf("set catalog view filter: %w", err)
	}

	if err := uc.repo.DeleteCityDraft(ctx, u.ID); err != nil {
		return fmt.Errorf("delete catalog city draft: %w", err)
	}

	competitions, hasMore, err := uc.repo.ListCatalogUpcoming(ctx, info.Today(u), &city, pageSize, 0)
	if err != nil {
		return fmt.Errorf("list catalog: %w", err)
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, listText(competitions, &city), listKeyboard(competitions, 0, hasMore, &city))
}

func (uc *UseCase) showPage(ctx context.Context, u *model.User, in dto.Input, page int) error {
	city, err := uc.resolveCity(ctx, u)
	if err != nil {
		return err
	}

	competitions, hasMore, err := uc.repo.ListCatalogUpcoming(ctx, info.Today(u), city, pageSize, page*pageSize)
	if err != nil {
		return fmt.Errorf("list catalog: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.EditMessageWithKeyboard(
		ctx, in.ChatID, in.MessageID, listText(competitions, city), listKeyboard(competitions, page, hasMore, city),
	)
}

func (uc *UseCase) promptCity(ctx context.Context, u *model.User, in dto.Input) error {
	if err := uc.repo.SetCityDraft(ctx, u.ID); err != nil {
		return fmt.Errorf("set catalog city draft: %w", err)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return uc.bot.SendWithKeyboard(ctx, in.ChatID, "В каком городе искать?\nНапример: Москва", cancelCityKeyboard())
}

// cancelCity backs out of the free-text city prompt without requiring any
// text at all — the draft is cleared and the catalog list reappears right
// in place of the prompt, same screen the "📍 ..." button was tapped from.
func (uc *UseCase) cancelCity(ctx context.Context, u *model.User, in dto.Input) error {
	if err := uc.repo.DeleteCityDraft(ctx, u.ID); err != nil {
		return fmt.Errorf("delete catalog city draft: %w", err)
	}

	return uc.showPage(ctx, u, in, 0)
}

func cancelCityKeyboard() dto.Keyboard {
	return dto.Keyboard{dto.Row(dto.Button{Label: "❌ Отмена", Data: callbackCityCancel})}
}

func (uc *UseCase) setCity(ctx context.Context, u *model.User, in dto.Input, city *string) error {
	if err := uc.repo.SetViewFilter(ctx, u.ID, city); err != nil {
		return fmt.Errorf("set catalog view filter: %w", err)
	}

	return uc.showPage(ctx, u, in, 0)
}

// resolveCity returns the explicit per-view override if one is active,
// otherwise falls back to u.City (which may itself be nil — "show every
// city" is the correct behavior either way, not an error).
func (uc *UseCase) resolveCity(ctx context.Context, u *model.User) (*string, error) {
	override, err := uc.repo.GetViewFilter(ctx, u.ID)
	if errors.Is(err, model.ErrNotFound) {
		return u.City, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get catalog view filter: %w", err)
	}

	return override.City, nil
}

func listText(competitions []*model.Competition, city *string) string {
	if len(competitions) == 0 {
		if city != nil {
			return fmt.Sprintf("🔎 Найти турнир\n\nВ городе «%s» пока нет опубликованных турниров.", *city)
		}

		return "🔎 Найти турнир\n\nПока нет опубликованных турниров."
	}

	return "🔎 Найти турнир"
}

func listKeyboard(competitions []*model.Competition, page int, hasMore bool, city *string) dto.Keyboard {
	kb := make(dto.Keyboard, 0, len(competitions)+3)

	for _, c := range competitions {
		kb = append(kb, dto.Row(dto.Button{Label: rowLabel(c), Data: fmt.Sprintf("catalog:view:%d", c.ID)}))
	}

	if nav := navRow(page, hasMore); len(nav) > 0 {
		kb = append(kb, nav)
	}

	kb = append(kb, dto.Row(dto.Button{Label: cityButtonLabel(city), Data: callbackCityPrompt}))

	if city != nil {
		kb = append(kb, dto.Row(dto.Button{Label: "Показать все города", Data: callbackCityAll}))
	}

	kb = append(kb, dto.Row(dto.Button{Label: "← Назад", Data: callbackCompetitionList}))

	return kb
}

func cityButtonLabel(city *string) string {
	if city != nil {
		return fmt.Sprintf("📍 %s — изменить", *city)
	}

	return "📍 Указать город"
}

func rowLabel(c *model.Competition) string {
	label := fmt.Sprintf("%s — %s", info.FormatDateRow(c.Date), c.Title)

	if c.City != nil && *c.City != "" {
		label += fmt.Sprintf(" (%s)", *c.City)
	}

	return label
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
