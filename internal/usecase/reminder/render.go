package reminder

import (
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/info"
)

const callbackDisable = "reminder:disable"

const disabledText = "🔕 Напоминания отключены."

// reminderText is deliberately plain, present-tense, gender-neutral and
// non-judgmental — no "Был/Пропустил", no streak-shaming. See the
// feature's own spec for why.
func reminderText(slot *model.ScheduleSlot) string {
	return fmt.Sprintf(
		"На этой неделе ещё не было тренировок. Сегодня по расписанию: %s в %s.",
		info.TrainingTypeLabel(slot.TrainingType), info.FormatTime(slot.TimeMinutes),
	)
}

func reminderKeyboard() dto.Keyboard {
	return dto.Keyboard{dto.Row(dto.Button{Label: "🔕 Отключить напоминания", Data: callbackDisable})}
}
