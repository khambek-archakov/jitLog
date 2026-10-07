package steps

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	dateQuestion  = "Когда он проходит?\nНапиши дату, например 15.11.2026 или просто 15.11.\nЕсли турнир идёт несколько дней, укажи первый"
	dateNotParsed = "Не получилось разобрать дату 🤔\nНапиши так: 15.11.2026 или 15.11"
)

type DateStep struct {
	bot  sender
	repo draftRepo
}

func NewDate(bot sender, repo draftRepo) *DateStep {
	return &DateStep{bot: bot, repo: repo}
}

func (s *DateStep) Handle(ctx context.Context, u *model.User, d *model.UserCompetitionDraft, in dto.Input) error {
	if in.HasCallback && in.CallbackData == callbackBack {
		return s.handleBack(ctx, d, in)
	}

	if in.HasCallback {
		return s.bot.AnswerCallback(ctx, in.CallbackID)
	}

	if !in.HasMessage {
		return nil
	}

	date, ok := parseDate(strings.TrimSpace(in.Text))
	if !ok {
		return s.bot.Send(ctx, in.ChatID, dateNotParsed)
	}

	return s.finish(ctx, u, d, in, date)
}

func (s *DateStep) handleBack(ctx context.Context, d *model.UserCompetitionDraft, in dto.Input) error {
	d.Step = model.UserCompetitionDraftStepAwaitingTitle

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update user competition draft: %w", err)
	}

	if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(ctx, in.ChatID, titleQuestion, titleKeyboard())
}

// finish is the wizard's terminal step — it creates the real
// user_competition row and clears the draft, rather than advancing to
// another question. Every other field (city, link, end date, result) is
// deliberately not asked here — the confirmation card below offers them
// as quick follow-up actions instead, keeping the create flow to exactly
// two questions: title, date.
func (s *DateStep) finish(ctx context.Context, u *model.User, d *model.UserCompetitionDraft, in dto.Input, date time.Time) error {
	if d.Title == nil {
		return fmt.Errorf("user competition draft %d has no title set", d.ID)
	}

	c, err := s.repo.CreateUserCompetition(ctx, d.UserID, *d.Title, date)
	if err != nil {
		return fmt.Errorf("create user competition: %w", err)
	}

	if err := s.repo.DeleteDraft(ctx, d.UserID); err != nil {
		return fmt.Errorf("delete user competition draft: %w", err)
	}

	today := info.Today(u)

	return s.bot.SendWithKeyboard(ctx, in.ChatID, confirmationText(c, today), info.Keyboard(c, today))
}

func dateKeyboard() dto.Keyboard {
	return dto.Keyboard{dto.Row(backButton(), cancelButton())}
}

// parseDate accepts "02.01.2006" or bare "02.01" (current year assumed) —
// never rolls a bare day.month back a year for looking "future", since
// future tournament dates are the common case here, not a typo.
func parseDate(text string) (time.Time, bool) {
	if t, err := time.Parse("02.01.2006", text); err == nil {
		return t, true
	}

	t, err := time.Parse("02.01", text)
	if err != nil {
		return time.Time{}, false
	}

	return t.AddDate(time.Now().Year(), 0, 0), true
}

func confirmationText(c *model.UserCompetition, today time.Time) string {
	return "✅ Турнир добавлен\n\n" + info.Body(c, today)
}
