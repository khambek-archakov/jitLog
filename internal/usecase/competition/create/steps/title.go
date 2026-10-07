package steps

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	titleQuestion  = "Как называется соревнование?\nНапример: Moscow Open 2026"
	titleNotParsed = "Не смог разобрать название 🤔\nПопробуй уложиться в 100 символов."
)

// callbackCompetitionAdd mirrors the competition list screen's own
// "➕ Добавить" button data — it's one of the two triggers that kick this
// flow off, so TitleStep (the first step) needs to recognize it.
const callbackCompetitionAdd = "competition:add"

// callbackCompetitionAddFromCatalog mirrors catalog/list's own "➕ Добавить"
// button data — the second trigger for this same flow, started from the
// "🔎 Найти соревнование" screen instead.
const callbackCompetitionAddFromCatalog = "competition:add:from_catalog"

type TitleStep struct {
	bot  sender
	repo draftRepo
}

func NewTitle(bot sender, repo draftRepo) *TitleStep {
	return &TitleStep{bot: bot, repo: repo}
}

func (s *TitleStep) Handle(ctx context.Context, _ *model.User, d *model.UserCompetitionDraft, in dto.Input) error {
	if in.HasCallback && (in.CallbackData == callbackCompetitionAdd || in.CallbackData == callbackCompetitionAddFromCatalog) {
		if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}

		return s.bot.SendWithKeyboard(ctx, in.ChatID, titleQuestion, titleKeyboard())
	}

	if in.HasCallback {
		return s.bot.AnswerCallback(ctx, in.CallbackID)
	}

	if !in.HasMessage {
		return nil
	}

	title, ok := parseTitle(in.Text)
	if !ok {
		return s.bot.Send(ctx, in.ChatID, titleNotParsed)
	}

	d.Title = &title
	d.Step = model.UserCompetitionDraftStepAwaitingDate

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update user competition draft: %w", err)
	}

	return s.bot.SendWithKeyboard(ctx, in.ChatID, dateQuestion, dateKeyboard())
}

// titleKeyboard's only button is labeled "← Назад" rather than "❌ Отмена"
// — this is the first question, so going back and cancelling the whole
// wizard are the same action. It still carries callbackCancel, so
// create.go's existing centralized cancel handling catches it unchanged.
func titleKeyboard() dto.Keyboard {
	return dto.Keyboard{dto.Row(dto.Button{Label: "← Назад", Data: callbackCancel})}
}

func parseTitle(text string) (string, bool) {
	title := strings.TrimSpace(text)

	n := utf8.RuneCountInString(title)
	if n < 1 || n > 100 {
		return "", false
	}

	return title, true
}
