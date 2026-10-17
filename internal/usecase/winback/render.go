package winback

import (
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	callbackHi  = "winback:hi"
	callbackOff = "winback:off"
)

// greetingText is deliberately just this — no "давно не заходил", no "мы
// скучаем", no mention of a lost streak. See the feature's own spec. Only
// the text fallback uses it as a caption/message body — the animation
// itself sends with an empty caption (see send).
const greetingText = "Привет 👋"

const declinedText = "Окей, без гифок."

// winbackKeyboard is two rows, one button each — shared by both the
// animation and its text fallback, so the two paths can never drift.
func winbackKeyboard() dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "Привет 👋", Data: callbackHi}),
		dto.Row(dto.Button{Label: "Давай только без гифок", Data: callbackOff}),
	}
}
