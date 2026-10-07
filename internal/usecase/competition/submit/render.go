package submit

import (
	"fmt"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// candidateKeyboard shows the "Это один из них?" picker — one button per
// date/city match, plus an explicit bypass for "none of these, make a new
// catalog entry" (never silently assumed).
func candidateKeyboard(personalID int64, candidates []*model.Competition) dto.Keyboard {
	kb := make(dto.Keyboard, 0, len(candidates)+1)

	for _, cand := range candidates {
		label := info.FormatDateRow(cand.Date) + " — " + cand.Title
		if cand.City != nil && *cand.City != "" {
			label += " (" + *cand.City + ")"
		}

		kb = append(kb, dto.Row(dto.Button{
			Label: label, Data: fmt.Sprintf("%s%d:attach:%d", callbackSubmitPrefix, personalID, cand.ID),
		}))
	}

	kb = append(kb, dto.Row(dto.Button{
		Label: "Нет, это новый турнир", Data: fmt.Sprintf("%s%d:new", callbackSubmitPrefix, personalID),
	}))

	return kb
}

func moderationText(c *model.Competition, url string) string {
	city := "—"
	if c.City != nil && *c.City != "" {
		city = *c.City
	}

	return fmt.Sprintf("📥 Новая заявка в каталог\n\n%s\n📅 %s\n🏙 %s\n🔗 %s", c.Title, info.FormatDate(c.Date), city, url)
}

func moderationKeyboard(competitionID int64) dto.Keyboard {
	prefix := fmt.Sprintf("competition:moderate:%d:", competitionID)

	return dto.Keyboard{
		dto.Row(
			dto.Button{Label: "✅ Одобрить", Data: prefix + "approve"},
			dto.Button{Label: "❌ Отклонить", Data: prefix + "reject"},
		),
		dto.Row(dto.Button{Label: "🔗 Это дубль", Data: prefix + "dup"}),
	}
}
