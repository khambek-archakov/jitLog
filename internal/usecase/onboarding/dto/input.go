package dto

// Input is the transport-neutral view of one incoming update that the
// onboarding usecase and its step handlers work with. Translating a real
// transport update (e.g. a Telegram update) into an Input is a delivery
// concern, not this usecase's — see internal/handler/update.
type Input struct {
	TelegramID int64
	ChatID     int64

	IsStartCmd bool

	HasMessage bool
	Text       string

	HasCallback  bool
	CallbackID   string
	CallbackData string
}
