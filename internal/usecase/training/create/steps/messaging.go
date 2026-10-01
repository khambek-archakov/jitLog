package steps

import (
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackBack is shared by every step after the first (date) — pressing it
// moves the draft one step back without discarding data already collected.
const callbackBack = "training:back"

// callbackCancel mirrors create's own private constant — every step shows
// it, but create.UseCase.dispatch handles it centrally (delete the draft,
// show the main menu) before a step ever sees it.
const callbackCancel = "training:cancel"

func backButton() dto.Button {
	return dto.Button{Label: "Назад", Data: callbackBack}
}

func cancelButton() dto.Button {
	return dto.Button{Label: "❌ Отмена", Data: callbackCancel}
}
