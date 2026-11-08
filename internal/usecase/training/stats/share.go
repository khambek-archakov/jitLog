package stats

import (
	"context"
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/share"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/usertime"
)

// callbackShareMonth is the "📤 Поделиться" button's own callback — only
// ever shown on the Month tab, and only once it has ≥1 training (see
// statsKeyboard). It always means "this month", so the callback itself
// carries no extra data.
const callbackShareMonth = "share:month"

const sharePhotoFilename = "jitlog-month.png"

const shareCaption = "Поделись своим прогрессом 🥋"

func shareKeyboard() dto.Keyboard {
	return dto.Keyboard{dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack})}
}

// handleShare renders the current month's recap image and sends it as
// its own new message — the stats screen underneath is left exactly as
// it was, never edited or removed. Nothing is stored or cached: a repeat
// tap just re-aggregates and re-renders from scratch.
func (uc *UseCase) handleShare(ctx context.Context, u *model.User, in dto.Input) error {
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
	c := countStats(filterByPeriod(all, periodMonth, now))

	if c.Total == 0 {
		// Stale tap — the button shouldn't have been visible, but a
		// message already sent to the user can always be tapped again
		// after the data it was built from changed (e.g. the month's
		// only training got deleted). Silently do nothing rather than
		// share an empty image.
		return nil
	}

	img, err := share.Render(share.Input{
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

	return uc.bot.SendPhotoWithKeyboard(ctx, in.ChatID, img, sharePhotoFilename, shareCaption, shareKeyboard())
}

func nameOrEmpty(u *model.User) string {
	if u.Name == nil {
		return ""
	}

	return *u.Name
}
