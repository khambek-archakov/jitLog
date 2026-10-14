package ajp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackModeratePrefix mirrors competition/moderate's own private
// constant — every button built here is handled there, unchanged,
// regardless of which message (this import's own, or a user's
// competition/submit) it was rendered into.
const callbackModeratePrefix = "competition:moderate:"

const (
	callbackApproveAll = callbackModeratePrefix + "ajp:approve_all"
	callbackReviewZero = callbackModeratePrefix + "ajp:review:0"
)

// notify sends every admin message this run's Report calls for — at most
// one for new competitions (shape depends on FirstRun/count, see
// sendNewCombined/sendNewBatchSummary), plus one per detected diff.
func (uc *UseCase) notify(ctx context.Context, report *Report) error {
	if len(report.Created) > 0 {
		if report.FirstRun || len(report.Created) > maxCombinedMessageItems {
			if err := uc.sendNewBatchSummary(ctx, report.Created); err != nil {
				return err
			}
		} else if err := uc.sendNewCombined(ctx, report.Created); err != nil {
			return err
		}
	}

	for _, d := range report.Diffs {
		if err := uc.sendDiffNotice(ctx, d); err != nil {
			return err
		}
	}

	return nil
}

// sendNewCombined is the default shape for a handful of new competitions
// (not the first run, 10 or fewer) — one message, with every new
// competition's own ✅/❌/🔗 row pair, exactly as competition/submit's own
// moderationKeyboard shape (duplicated here rather than imported, per this
// project's own small-interface-per-consumer convention — moderate.Handle
// dispatches these identically regardless of where the message came from).
func (uc *UseCase) sendNewCombined(ctx context.Context, created []*model.Competition) error {
	var text strings.Builder

	fmt.Fprintf(&text, "AJP: %d новых соревнований на проверке", len(created))

	kb := make(dto.Keyboard, 0, len(created)*2)

	for _, c := range created {
		text.WriteString("\n\n")
		text.WriteString(competitionLine(c))

		prefix := fmt.Sprintf("%s%d:", callbackModeratePrefix, c.ID)
		kb = append(
			kb,
			dto.Row(
				dto.Button{Label: "✅ Одобрить", Data: prefix + "approve"},
				dto.Button{Label: "❌ Отклонить", Data: prefix + "reject"},
			),
			dto.Row(dto.Button{Label: "🔗 Это дубль", Data: prefix + "dup"}),
		)
	}

	uc.appendSourceLabel(&text)

	return uc.bot.SendWithKeyboard(ctx, uc.adminChatID, text.String(), kb)
}

// sendNewBatchSummary is the first-run/>10 shape — a short list plus
// "✅ Одобрить все" / "👀 Смотреть по одному", both handled by
// competition/moderate against whatever is pending and author-less live
// at the moment they're tapped (not frozen to this run's own batch).
func (uc *UseCase) sendNewBatchSummary(ctx context.Context, created []*model.Competition) error {
	var text strings.Builder

	fmt.Fprintf(&text, "AJP: %d новых соревнований на проверке", len(created))

	for _, c := range created {
		fmt.Fprintf(&text, "\n• %s — %s", c.Title, dateLabel(c))
	}

	uc.appendSourceLabel(&text)

	kb := dto.Keyboard{
		dto.Row(dto.Button{Label: "✅ Одобрить все", Data: callbackApproveAll}),
		dto.Row(dto.Button{Label: "👀 Смотреть по одному", Data: callbackReviewZero}),
	}

	return uc.bot.SendWithKeyboard(ctx, uc.adminChatID, text.String(), kb)
}

// appendSourceLabel adds "Источник: файл X" under a new-competitions
// message when this run read from a file (see SetSourceLabel) — a diff
// notice never gets this line, only new-competition messages do.
func (uc *UseCase) appendSourceLabel(text *strings.Builder) {
	if uc.sourceLabel == "" {
		return
	}

	fmt.Fprintf(text, "\n\nИсточник: %s", uc.sourceLabel)
}

func (uc *UseCase) sendDiffNotice(ctx context.Context, d DiffNotice) error {
	oldCity := "—"
	if d.Competition.City != nil && *d.Competition.City != "" {
		oldCity = *d.Competition.City
	}

	text := fmt.Sprintf(
		"AJP: расхождение в соревновании\n\n%s\n\nБыло: %s, %s\nСтало: %s, %s",
		d.Competition.Title,
		info.FormatDateRangeCompact(d.Competition.Date, endOrDate(d.Competition.Date, d.Competition.EndDate)), oldCity,
		info.FormatDateRangeCompact(d.ProposedDate, endOrDate(d.ProposedDate, d.ProposedEndDate)), d.ProposedCity,
	)

	prefix := fmt.Sprintf("%sajp:diff:%d:", callbackModeratePrefix, d.Competition.ID)
	kb := dto.Keyboard{
		dto.Row(
			dto.Button{Label: "✏️ Обновить", Data: prefix + "apply"},
			dto.Button{Label: "➡️ Оставить", Data: prefix + "keep"},
		),
	}

	return uc.bot.SendWithKeyboard(ctx, uc.adminChatID, text, kb)
}

func competitionLine(c *model.Competition) string {
	city := "—"
	if c.City != nil && *c.City != "" {
		city = *c.City
	}

	return fmt.Sprintf("%s\n📅 %s 🏙 %s", c.Title, dateLabel(c), city)
}

func dateLabel(c *model.Competition) string {
	return info.FormatDateRangeCompact(c.Date, endOrDate(c.Date, c.EndDate))
}

func endOrDate(date time.Time, endDate *time.Time) time.Time {
	if endDate != nil {
		return *endDate
	}

	return date
}
