package steps

import (
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// callbackBack is shared by every step after the first (date) — pressing it
// moves the draft one step back without discarding data already collected.
const callbackBack = "training:back"

func backButton() dto.Button {
	return dto.Button{Label: "Назад", Data: callbackBack}
}
