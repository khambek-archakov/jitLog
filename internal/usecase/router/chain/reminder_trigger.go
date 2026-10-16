package chain

import (
	"context"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const callbackReminderPrefix = "reminder:"

// reminderTrigger covers the "🔕 Отключить напоминания" button on whatever
// standalone reminder message internal/usecase/reminder's own ticker sent
// — the only reminder:*-prefixed callback that ever reaches the bot.
type reminderTrigger struct {
	reminder reminderDisable
}

func (h *reminderTrigger) Handle(ctx context.Context, u *model.User, in dto.Input) error {
	if !in.HasCallback || !strings.HasPrefix(in.CallbackData, callbackReminderPrefix) {
		return model.ErrSkip
	}

	return h.reminder.Handle(ctx, u, in)
}
