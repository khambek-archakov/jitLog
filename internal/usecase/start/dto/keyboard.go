package dto

// Button is one inline keyboard button: the label shown to the user plus
// the callback data it carries back when tapped.
type Button struct {
	Label string
	Data  string
}

// Keyboard is an inline keyboard laid out as rows of buttons.
type Keyboard [][]Button

// Row is a convenience constructor for a single-row slice of buttons,
// e.g. Row(Button{"Назад", callbackBack}).
func Row(buttons ...Button) []Button {
	return buttons
}
