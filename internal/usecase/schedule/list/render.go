package list

import (
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/schedule/info"
)

// callbackScheduleViewPrefix mirrors internal/usecase/schedule/info's own
// private constant — tapping a slot row hands off to info.
const callbackScheduleViewPrefix = "schedule:view:"

func listText(slots []*model.ScheduleSlot) string {
	if len(slots) == 0 {
		return "📅 Расписание\n\nПока нет ни одного слота — добавь первый, и здесь появится твоя неделя."
	}

	return "📅 Расписание"
}

func listKeyboard(slots []*model.ScheduleSlot) dto.Keyboard {
	kb := make(dto.Keyboard, 0, len(slots)+2)

	for _, s := range slots {
		kb = append(kb, dto.Row(dto.Button{Label: info.Body(s), Data: fmt.Sprintf("%s%d", callbackScheduleViewPrefix, s.ID)}))
	}

	kb = append(kb, dto.Row(dto.Button{Label: "➕ Добавить", Data: callbackScheduleAdd}))
	kb = append(kb, dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack}))

	return kb
}
