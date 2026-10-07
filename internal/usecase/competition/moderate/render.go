package moderate

import (
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

func mergeCandidateKeyboard(pendingID int64, candidates []*model.Competition) dto.Keyboard {
	kb := make(dto.Keyboard, 0, len(candidates))

	for _, c := range candidates {
		label := info.FormatDateRow(c.Date) + " — " + c.Title
		if c.City != nil && *c.City != "" {
			label += " (" + *c.City + ")"
		}

		kb = append(kb, dto.Row(dto.Button{
			Label: label, Data: fmt.Sprintf("%s%d:merge:%d", callbackModeratePrefix, pendingID, c.ID),
		}))
	}

	return kb
}
