package dto

// Input is the transport-neutral view of one incoming update that the
// onboarding usecase and its step handlers work with. Translating a real
// transport update (e.g. a Telegram update) into an Input is a delivery
// concern, not this usecase's — see internal/handler/update.
type Input struct {
	TelegramID int64
	ChatID     int64
	MessageID  int

	IsStartCmd bool
	// StartPayload is /start's own deep-link payload (the text after
	// "/start "), unvalidated — only ever set alongside IsStartCmd.
	StartPayload string

	HasMessage bool
	Text       string

	HasCallback  bool
	CallbackID   string
	CallbackData string
}
