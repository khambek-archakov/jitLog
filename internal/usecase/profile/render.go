package profile

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
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

	return fmt.Sprintf("👤 Профиль\n\nИмя: %s\nВозраст: %s\nПояс: %s", name, age, beltLabel(u.Belt))
}

func profileKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "✏️ Редактировать", Data: callbackProfileEdit}),
		dto.Row(dto.Button{Label: "← Главное меню", Data: callbackMenuBack}),
	}
}

const editMenuText = "✏️ Что изменить?"

func editMenuKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "🥋 Пояс", Data: callbackProfileBeltEdit}),
		dto.Row(dto.Button{Label: "🎂 Возраст", Data: callbackProfileAgeEdit}),
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

const beltPickerText = "✏️ Выбери новый пояс:"

// beltPickerKeyboard offers every belt except the one the user already
// has — "changing" to your current belt makes no sense, and belt_promotion
// only allows recording each belt once per user anyway.
func beltPickerKeyboard(current model.Belt) dto.Keyboard {
	all := []model.Belt{model.BeltWhite, model.BeltBlue, model.BeltPurple, model.BeltBrown, model.BeltBlack}

	var buttons []dto.Button

	for _, b := range all {
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
