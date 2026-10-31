package dto

// Command is a bot command registered with Telegram's own "/" picker
// (see Gateway.SetMyCommands) — Name without its leading slash.
type Command struct {
	Name        string
	Description string
}
