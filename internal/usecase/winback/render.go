package winback

import (
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	callbackHi  = "winback:hi"
	callbackOff = "winback:off"
)

// greetingText is deliberately just this — no "давно не заходил", no "мы
// скучаем", no mention of a lost streak. See the feature's own spec.
const greetingText = "Привет!"

const declinedText = "Окей, без гифок."

func winbackKeyboard() dto.Keyboard {
	return dto.Keyboard{dto.Row(
		dto.Button{Label: "Привет!", Data: callbackHi},
		dto.Button{Label: "Вот давай без гифок", Data: callbackOff},
	)}
}
