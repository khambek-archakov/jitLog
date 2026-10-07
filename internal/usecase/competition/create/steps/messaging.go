package steps

import "github.com/khambek-archakov/jitLog/internal/usecase/dto"

// callbackCancel mirrors create's own private constant — create.go handles
// it centrally, steps only need the literal to render the button.
const callbackCancel = "competition:draft:cancel"

// callbackBack moves the draft one step back without discarding data
// already collected — handled per-step (unlike Cancel), since each step
// has a different "previous" question.
const callbackBack = "competition:draft:back"

func cancelButton() dto.Button {
	return dto.Button{Label: "❌ Отмена", Data: callbackCancel}
}

func backButton() dto.Button {
	return dto.Button{Label: "← Назад", Data: callbackBack}
}
