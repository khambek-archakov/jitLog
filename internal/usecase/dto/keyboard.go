package dto

// Button is one inline keyboard button. Exactly one of Data/URL is set:
// Data carries callback data back when tapped, URL opens a link directly
// (Telegram handles it client-side — no update is ever sent back to the
// bot for a URL button).
type Button struct {
	Label string
	Data  string
	URL   string
}

// Keyboard is an inline keyboard laid out as rows of buttons.
type Keyboard [][]Button

// Row is a convenience constructor for a single-row slice of buttons,
// e.g. Row(Button{"Назад", callbackBack}).
func Row(buttons ...Button) []Button {
	return buttons
}
