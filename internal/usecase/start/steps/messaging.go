package steps

import (
	"github.com/khambek-archakov/jitLog/internal/usecase/start/dto"
)

const callbackBack = "start:back"

func backButton() dto.Button {
	return dto.Button{Label: "Назад", Data: callbackBack}
}
