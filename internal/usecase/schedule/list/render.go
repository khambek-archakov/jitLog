package list

import (
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/timezone"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/info"
)

// callbackScheduleViewPrefix mirrors internal/usecase/schedule/info's own
// private constant — tapping a slot row hands off to info.
const callbackScheduleViewPrefix = "schedule:view:"

func listText(slots []*model.ScheduleSlot, u *model.User) string {
	if len(slots) == 0 {
		return "📅 Расписание\n\nПока нет ни одного слота — добавь первый, и здесь появится твоя неделя."
	}

	return "📅 Расписание\n\n" + reminderStatusText(u)
}

// reminderStatusText is only ever shown once at least one slot exists —
// see listKeyboard, which gates the toggle button on the exact same
// condition.
func reminderStatusText(u *model.User) string {
	if !u.RemindersEnabled {
		return "🔔 Напоминания выключены"
	}

	label := "—"

	if u.Timezone != nil {
		if l, ok := timezone.LabelForIANA(*u.Timezone); ok {
			label = l
		}
	}

	return "🔔 Напоминания включены. Часовой пояс: " + label
}

func listKeyboard(slots []*model.ScheduleSlot, u *model.User) dto.Keyboard {
	kb := make(dto.Keyboard, 0, len(slots)+3)

	for _, s := range slots {
		kb = append(kb, dto.Row(dto.Button{Label: info.Body(s), Data: fmt.Sprintf("%s%d", callbackScheduleViewPrefix, s.ID)}))
	}

	kb = append(kb, dto.Row(dto.Button{Label: "➕ Добавить", Data: callbackScheduleAdd}))

	if len(slots) > 0 {
		kb = append(kb, dto.Row(dto.Button{Label: reminderToggleLabel(u), Data: callbackReminderToggle}))
	}

	kb = append(kb, dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack}))

	return kb
}

func reminderToggleLabel(u *model.User) string {
	if u.RemindersEnabled {
		return "🔔 Напоминания: вкл"
	}

	return "🔔 Напоминания: выкл"
}

const tzPickerText = "Чтобы напоминание приходило вовремя, выбери свой часовой пояс:"

func tzPickerKeyboard() dto.Keyboard {
	kb := timezone.Rows(callbackTZSetPrefix)

	return append(kb, dto.Row(dto.Button{Label: "Отмена", Data: callbackTZCancel}))
}
