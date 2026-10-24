package reminder

import (
	"fmt"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	scheduleinfo "github.com/khambek-archakov/jitLog/internal/usecase/schedule/info"
)

const callbackDisable = "reminder:disable"

const disabledText = "🔕 Напоминания отключены."

// reminderText is deliberately plain, present-tense, gender-neutral and
// non-judgmental — no "Был/Пропустил", no streak-shaming. See the
// feature's own spec for why.
func reminderText(slot *model.ScheduleSlot) string {
	return fmt.Sprintf(
		"На этой неделе ещё не было тренировок. Сегодня по расписанию: %s в %s.",
		scheduleinfo.TrainingTypeLabel(slot.TrainingType), scheduleinfo.FormatTime(slot.TimeMinutes),
	)
}

func reminderKeyboard() dto.Keyboard {
	return dto.Keyboard{dto.Row(dto.Button{Label: "🔕 Отключить напоминания", Data: callbackDisable})}
}

const callbackRecPrefix = "rec:"

const recDateLayout = "20060102"

// postSessionText — same tone rules as reminderText: no "Пропустил", no
// guilt. isYesterday only ever flips to true once the slot's own send
// window has been pushed past midnight by the 09:00/22:00 clamp (see
// clampPostSessionSendAt) — the slot's own calendar date D no longer
// matches the local date the message actually goes out on.
func postSessionText(slot *model.ScheduleSlot, isYesterday bool) string {
	dayPrefix := ""
	if isYesterday {
		dayPrefix = "Вчера, "
	}

	return fmt.Sprintf(
		"%s, %s%s. Записать тренировку?",
		scheduleinfo.TrainingTypeLabel(slot.TrainingType), dayPrefix, scheduleinfo.FormatTime(slot.TimeMinutes),
	)
}

func postSessionKeyboard(slot *model.ScheduleSlot, d time.Time) dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{
			Label: fmt.Sprintf("✅ Записать (%d мин)", slot.DurationMinutes),
			Data:  fmt.Sprintf("%s%d:%s", callbackRecPrefix, slot.ID, d.Format(recDateLayout)),
		}),
		dto.Row(dto.Button{Label: "🔕 Отключить напоминания", Data: callbackDisable}),
	}
}

func parseRecDate(s string) (time.Time, bool) {
	t, err := time.ParseInLocation(recDateLayout, s, time.UTC)
	if err != nil {
		return time.Time{}, false
	}

	return t, true
}
