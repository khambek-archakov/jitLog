package steps

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/info"
)

const dayQuestion = "📅 Какой день недели?"

// callbackScheduleAdd mirrors the schedule list screen's own "➕ Добавить"
// button data (see internal/usecase/schedule/list and
// internal/usecase/router/chain) — it's the trigger that kicks this flow
// off, so DayStep (the first step) needs to recognize it too.
const callbackScheduleAdd = "schedule:add"

// callbackDayPrefix is followed by a weekday number, 1=Monday..7=Sunday.
const callbackDayPrefix = "schedule:draft:day:"

type DayStep struct {
	bot  sender
	repo draftRepo
}

func NewDay(bot sender, repo draftRepo) *DayStep {
	return &DayStep{bot: bot, repo: repo}
}

func (s *DayStep) Handle(ctx context.Context, d *model.ScheduleDraft, in dto.Input) error {
	if in.HasCallback && in.CallbackData == callbackScheduleAdd {
		if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
			return err
		}

		return s.bot.SendWithKeyboard(in.ChatID, dayQuestion, dayKeyboard())
	}

	if !in.HasCallback {
		return nil
	}

	day, ok := dayFromCallback(in.CallbackData)
	if !ok {
		return s.bot.AnswerCallback(in.CallbackID)
	}

	d.DayOfWeek = &day
	d.Step = model.ScheduleDraftStepAwaitingTime

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update schedule draft: %w", err)
	}

	if err := s.bot.AnswerCallback(in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(in.ChatID, timeQuestion, timeKeyboard())
}

func dayKeyboard() dto.Keyboard {
	buttons := make([]dto.Button, 0, 7)
	for day := int16(1); day <= 7; day++ {
		buttons = append(buttons, dto.Button{Label: info.DayLabel(day), Data: dayCallback(day)})
	}

	return dto.Keyboard{
		buttons,
		dto.Row(cancelButton()),
	}
}

func dayCallback(day int16) string {
	return fmt.Sprintf("%s%d", callbackDayPrefix, day)
}

func dayFromCallback(data string) (int16, bool) {
	if !strings.HasPrefix(data, callbackDayPrefix) {
		return 0, false
	}

	n, err := strconv.Atoi(strings.TrimPrefix(data, callbackDayPrefix))
	if err != nil || n < 1 || n > 7 {
		return 0, false
	}

	return int16(n), true
}
