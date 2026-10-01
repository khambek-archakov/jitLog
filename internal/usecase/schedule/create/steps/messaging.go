package steps

import "github.com/khambek-archakov/jitLog/internal/usecase/dto"

// callbackBack is shared by every step after the first (day) — pressing it
// moves the draft one step back without discarding data already collected.
const callbackBack = "schedule:draft:back"

// callbackCancel mirrors create's own private constant — create.go handles
// it centrally, steps only need the literal to render the button.
const callbackCancel = "schedule:draft:cancel"

func backButton() dto.Button {
	return dto.Button{Label: "Назад", Data: callbackBack}
}

func cancelButton() dto.Button {
	return dto.Button{Label: "❌ Отмена", Data: callbackCancel}
}
