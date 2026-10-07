package steps

import "github.com/khambek-archakov/jitLog/internal/usecase/dto"

// callbackCancel mirrors create's own private constant — create.go handles
// it centrally, steps only need the literal to render the button. No
// "Назад" button exists in this wizard (unlike training/schedule's) — it's
// deliberately just two questions, so there's nothing worth stepping back
// through.
const callbackCancel = "competition:draft:cancel"

func cancelButton() dto.Button {
	return dto.Button{Label: "❌ Отмена", Data: callbackCancel}
}
