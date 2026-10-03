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

const (
	timeQuestion  = "🕐 Во сколько?"
	timeNotParsed = "Не смог разобрать время, напиши в формате ЧЧ:ММ, например: 18:30."
)

const (
	callbackTimePresetPrefix = "schedule:draft:time:preset:"
	callbackTimeOther        = "schedule:draft:time:other"
)

// presetTimes are offered as quick picks, in minutes since midnight.
var presetTimes = [...]int16{11 * 60, 12 * 60, 19 * 60, 19*60 + 30, 21 * 60}

type TimeStep struct {
	bot  sender
	repo draftRepo
}

func NewTime(bot sender, repo draftRepo) *TimeStep {
	return &TimeStep{bot: bot, repo: repo}
}

func (s *TimeStep) Handle(ctx context.Context, d *model.ScheduleDraft, in dto.Input) error {
	if in.HasCallback {
		switch {
		case in.CallbackData == callbackBack:
			return s.handleBack(ctx, d, in)

		case in.CallbackData == callbackTimeOther:
			if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
				return err
			}

			return s.bot.Send(ctx, in.ChatID, "Напиши время в формате ЧЧ:ММ, например: 18:30")

		case strings.HasPrefix(in.CallbackData, callbackTimePresetPrefix):
			minutes, ok := parseTimeToken(strings.TrimPrefix(in.CallbackData, callbackTimePresetPrefix))
			if !ok {
				return s.bot.AnswerCallback(ctx, in.CallbackID)
			}

			return s.saveTime(ctx, d, in, minutes)

		default:
			return s.bot.AnswerCallback(ctx, in.CallbackID)
		}
	}

	if !in.HasMessage {
		return nil
	}

	minutes, ok := parseTimeText(strings.TrimSpace(in.Text))
	if !ok {
		return s.bot.Send(ctx, in.ChatID, timeNotParsed)
	}

	return s.saveTime(ctx, d, in, minutes)
}

func (s *TimeStep) saveTime(ctx context.Context, d *model.ScheduleDraft, in dto.Input, minutes int16) error {
	d.TimeMinutes = &minutes
	d.Step = model.ScheduleDraftStepAwaitingType

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update schedule draft: %w", err)
	}

	if in.HasCallback {
		if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
			return err
		}
	}

	return s.bot.SendWithKeyboard(ctx, in.ChatID, typeQuestion, typeKeyboard())
}

func (s *TimeStep) handleBack(ctx context.Context, d *model.ScheduleDraft, in dto.Input) error {
	d.Step = model.ScheduleDraftStepAwaitingDay

	if err := s.repo.UpdateDraft(ctx, d); err != nil {
		return fmt.Errorf("update schedule draft: %w", err)
	}

	if err := s.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	return s.bot.SendWithKeyboard(ctx, in.ChatID, dayQuestion, dayKeyboard())
}

func timeKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(presetButton(presetTimes[0]), presetButton(presetTimes[1]), presetButton(presetTimes[2])),
		dto.Row(presetButton(presetTimes[3]), presetButton(presetTimes[4])),
		dto.Row(dto.Button{Label: "Другое", Data: callbackTimeOther}),
		dto.Row(backButton(), cancelButton()),
	}
}

func presetButton(minutes int16) dto.Button {
	return dto.Button{Label: info.FormatTime(minutes), Data: callbackTimePresetPrefix + formatTimeToken(minutes)}
}

func formatTimeToken(minutes int16) string {
	return fmt.Sprintf("%02d%02d", minutes/60, minutes%60)
}

func parseTimeToken(s string) (int16, bool) {
	if len(s) != 4 {
		return 0, false
	}

	h, err1 := strconv.Atoi(s[:2])
	m, err2 := strconv.Atoi(s[2:])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}

	return int16(h*60 + m), true
}

func parseTimeText(s string) (int16, bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return 0, false
	}

	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}

	return int16(h*60 + m), true
}
