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
	titleQuestion  = "Как называется турнир?"
	titleNotParsed = "Название должно быть от 1 до 100 символов, напиши ещё раз."
)

// callbackCompetitionAdd mirrors the competition list screen's own
// "➕ Добавить турнир" button data — it's the trigger that kicks this flow
// off, so TitleStep (the first step) needs to recognize it too.
const callbackCompetitionAdd = "competition:add"

type TitleStep struct {
	bot  sender
	repo draftRepo
}

func NewTitle(bot sender, repo draftRepo) *TitleStep {
	return &TitleStep{bot: bot, repo: repo}
}

func (s *TitleStep) Handle(ctx context.Context, _ *model.User, d *model.UserCompetitionDraft, in dto.Input) error {
	if in.HasCallback && in.CallbackData == callbackCompetitionAdd {
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

func titleKeyboard() dto.Keyboard {
	return dto.Keyboard{dto.Row(cancelButton())}
}

func parseTitle(text string) (string, bool) {
	title := strings.TrimSpace(text)

	n := utf8.RuneCountInString(title)
	if n < 1 || n > 100 {
		return "", false
	}

	return title, true
}
