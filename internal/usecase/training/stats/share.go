package stats

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/share"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/usertime"
)

// callbackShareMonth/callbackShareYear are the "↗️ Поделиться" button's
// own callbacks — shown only on the Month and Year tabs, and only once
// that period has ≥1 training (see statsKeyboard). Each carries no
// extra data: it always means "this period, right now".
const (
	callbackSharePrefix = "share:"
	callbackShareMonth  = callbackSharePrefix + "month"
	callbackShareYear   = callbackSharePrefix + "year"
)

// shareCallback is statsKeyboard's own share button target for p —
// callbackShareMonth for every period except periodYear.
func shareCallback(p period) string {
	if p == periodYear {
		return callbackShareYear
	}

	return callbackShareMonth
}

// sharePhotoFilename names the sent photo by period — "jitlog-month.png"
// unchanged for the Month tab, "jitlog-year.png" for the new Year one.
func sharePhotoFilename(p period) string {
	if p == periodYear {
		return "jitlog-year.png"
	}

	return "jitlog-month.png"
}

func shareKeyboard() dto.Keyboard {
	return dto.Keyboard{dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack})}
}

// parseSharePeriod maps a share:* callback back to the period it means.
func parseSharePeriod(data string) (period, bool) {
	switch data {
	case callbackShareMonth:
		return periodMonth, true
	case callbackShareYear:
		return periodYear, true
	default:
		return "", false
	}
}

// handleShare renders the requested period's recap image and sends it
// as its own new message — the stats screen underneath is left exactly
// as it was, never edited or removed. Nothing is stored or cached: a
// repeat tap just re-aggregates and re-renders from scratch.
func (uc *UseCase) handleShare(ctx context.Context, u *model.User, in dto.Input) error {
	p, ok := parseSharePeriod(in.CallbackData)
	if !ok {
		return uc.bot.AnswerCallback(ctx, in.CallbackID)
	}

	if err := uc.bot.AnswerCallback(ctx, in.CallbackID); err != nil {
		return err
	}

	// Best-effort UX only — a failure here must never block the actual
	// photo that follows.
	_ = uc.bot.SendUploadPhotoAction(ctx, in.ChatID)

	all, err := uc.repo.ListAllTrainings(ctx, u.ID)
	if err != nil {
		return fmt.Errorf("list all trainings: %w", err)
	}

	now := uc.now().In(usertime.Location(u))
	c := countStats(filterByPeriod(all, p, now))

	if c.Total == 0 {
		// Stale tap — the button shouldn't have been visible, but a
		// message already sent to the user can always be tapped again
		// after the data it was built from changed (e.g. the period's
		// only training got deleted). Silently do nothing rather than
		// share an empty image.
		return nil
	}

	sharePeriod := share.PeriodMonth
	if p == periodYear {
		sharePeriod = share.PeriodYear
	}

	img, err := share.Render(share.Input{
		Period:       sharePeriod,
		Name:         nameOrEmpty(u),
		Month:        now,
		Trainings:    c.Total,
		TotalMinutes: c.TotalMinutes,
		Rounds:       c.TotalRounds,
		GiCount:      c.GiCount,
		NoGiCount:    c.NoGiCount,
		OpenMatCount: c.OpenMatCount,
	})
	if err != nil {
		return fmt.Errorf("render share image: %w", err)
	}

	// No caption — just the image.
	return uc.bot.SendPhotoWithKeyboard(ctx, in.ChatID, img, sharePhotoFilename(p), "", shareKeyboard())
}

func nameOrEmpty(u *model.User) string {
	if u.Name == nil {
		return ""
	}

	return *u.Name
}
