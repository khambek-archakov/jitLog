package profile

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/internal/timezone"
)

// callbackMenuBack mirrors internal/usecase/onboarding/steps' own private
// constant.
const callbackMenuBack = "menu:back"

func profileText(u *model.User) string {
	name := ""
	if u.Name != nil {
		name = *u.Name
	}

	age := "—"
	if u.Age != nil {
		age = fmt.Sprintf("%d", *u.Age)
	}

	text := fmt.Sprintf("👤 Профиль\n\nИмя: %s\nВозраст: %s\nПояс: %s", name, age, beltLabel(u.Belt))

	if u.Timezone == nil {
		return text
	}

	label := "—"
	if l, ok := timezone.LabelForIANA(*u.Timezone); ok {
		label = l
	}

	// Reminders on/off status lives on the "📅 Расписание" screen, not
	// here — see internal/usecase/schedule/list.
	return fmt.Sprintf("%s\nЧасовой пояс: %s", text, label)
}

func profileKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "✏️ Редактировать", Data: callbackProfileEdit}),
		dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack}),
	}
}

const tzPickerText = "Выбери свой часовой пояс:"

// tzPickerKeyboard's own "← Назад" reuses callbackProfileEdit directly
// (same as beltPickerKeyboard's) — picking a zone is stateless, so there's
// no draft to clean up on the way back, just like belt.
func tzPickerKeyboard() dto.Keyboard {
	kb := timezone.Rows(callbackProfileTZSetPrefix)

	return append(kb, dto.Row(dto.Button{Label: "← Назад", Data: callbackProfileEdit}))
}

const editMenuText = "✏️ Что изменить?"

func editMenuKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "🥋 Пояс", Data: callbackProfileBeltEdit}),
		dto.Row(dto.Button{Label: "🎂 Возраст", Data: callbackProfileAgeEdit}),
		dto.Row(dto.Button{Label: "🕐 Часовой пояс", Data: callbackProfileTZEdit}),
		dto.Row(dto.Button{Label: "← Назад", Data: callbackProfileShow}),
	}
}

const (
	askAgeText       = "Сколько тебе лет?"
	ageNotParsedText = "Не смог разобрать число, напиши возраст цифрами, например: 25."
)

// parseAge accepts a plain positive integer in a sane human-age range —
// loose enough not to reject real answers, tight enough to catch an
// obviously mistyped reply.
func parseAge(text string) (int16, bool) {
	age, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil || age <= 0 || age > 120 {
		return 0, false
	}

	return int16(age), true
}

// beltOrder is the belt progression, lowest first — both
// beltPickerKeyboard and beltRank (the belt-change congratulation's own
// "is this exactly one step up" check) share this one ordering.
var beltOrder = []model.Belt{model.BeltWhite, model.BeltBlue, model.BeltPurple, model.BeltBrown, model.BeltBlack}

// beltRank is belt's index in beltOrder, or -1 for an unrecognized belt
// (including model.BeltNone) — never matches any real belt's rank, so a
// transition away from "no belt yet" always counts as a promotion.
func beltRank(b model.Belt) int {
	for i, x := range beltOrder {
		if x == b {
			return i
		}
	}

	return -1
}

const beltPickerText = "✏️ Выбери новый пояс:"

// beltPickerKeyboard offers every belt except the one the user already
// has — "changing" to your current belt makes no sense, and belt_promotion
// only allows recording each belt once per user anyway.
func beltPickerKeyboard(current model.Belt) dto.Keyboard {
	var buttons []dto.Button

	for _, b := range beltOrder {
		if b == current {
			continue
		}

		buttons = append(buttons, dto.Button{Label: beltLabel(b), Data: callbackProfileBeltSetPrefix + beltToken(b)})
	}

	kb := make(dto.Keyboard, 0, len(buttons)+1)
	for _, b := range buttons {
		kb = append(kb, dto.Row(b))
	}

	kb = append(kb, dto.Row(dto.Button{Label: "← Назад", Data: callbackProfileEdit}))

	return kb
}

func beltLabel(b model.Belt) string {
	switch b {
	case model.BeltWhite:
		return "⚪ Белый"
	case model.BeltBlue:
		return "🔵 Синий"
	case model.BeltPurple:
		return "🟣 Пурпурный"
	case model.BeltBrown:
		return "🟤 Коричневый"
	case model.BeltBlack:
		return "⚫ Чёрный"
	default:
		return "—"
	}
}

// beltInstrumental gives belt's instrumental-case form ("с {X} поясом"),
// used by the belt-change congratulation's first line.
func beltInstrumental(b model.Belt) string {
	switch b {
	case model.BeltWhite:
		return "белым"
	case model.BeltBlue:
		return "синим"
	case model.BeltPurple:
		return "пурпурным"
	case model.BeltBrown:
		return "коричневым"
	case model.BeltBlack:
		return "чёрным"
	default:
		return string(b)
	}
}

// beltPrepositional gives belt's prepositional-case form ("на {X} поясе"),
// used by the congratulation's second line.
func beltPrepositional(b model.Belt) string {
	switch b {
	case model.BeltWhite:
		return "белом"
	case model.BeltBlue:
		return "синем"
	case model.BeltPurple:
		return "пурпурном"
	case model.BeltBrown:
		return "коричневом"
	case model.BeltBlack:
		return "чёрном"
	default:
		return string(b)
	}
}

// trainingWord picks the right Russian plural form of "тренировка" for n
// — same n%10/n%100 pattern as training/info's roundsWord and
// training/stats's weeksWord.
func trainingWord(n int) string {
	if n%100 >= 11 && n%100 <= 14 {
		return "тренировок"
	}

	switch n % 10 {
	case 1:
		return "тренировка"
	case 2, 3, 4:
		return "тренировки"
	default:
		return "тренировок"
	}
}

func beltToken(b model.Belt) string {
	switch b {
	case model.BeltWhite:
		return "white"
	case model.BeltBlue:
		return "blue"
	case model.BeltPurple:
		return "purple"
	case model.BeltBrown:
		return "brown"
	case model.BeltBlack:
		return "black"
	default:
		return ""
	}
}

func beltFromToken(token string) (model.Belt, bool) {
	switch token {
	case "white":
		return model.BeltWhite, true
	case "blue":
		return model.BeltBlue, true
	case "purple":
		return model.BeltPurple, true
	case "brown":
		return model.BeltBrown, true
	case "black":
		return model.BeltBlack, true
	default:
		return model.BeltNone, false
	}
}
