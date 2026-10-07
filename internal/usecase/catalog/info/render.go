package info

import (
	"fmt"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
	competitioninfo "github.com/khambek-archakov/jitLog/internal/usecase/competition/info"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

const (
	callbackAddPrefix = "catalog:add:"
	// callbackCatalogList mirrors internal/usecase/catalog/list's own
	// private constant — the card's "← Назад" button goes back there.
	callbackCatalogList = "catalog:list"
)

// Card is a catalog entry's read-only view — unlike a personal record's
// card, there's no edit/delete here (only the admin, via
// competition/moderate, ever changes a catalog entry), just the one
// "➕ В мои" action.
func Card(c *model.Competition, url *string, today time.Time) string {
	body := c.Title + "\n\n📅 " + competitioninfo.DateRange(c.Date, c.EndDate)

	if c.City != nil && *c.City != "" {
		body += "\n🏙 " + *c.City
	}

	body += "\n" + competitioninfo.Status(c.Date, c.EndDate, today)

	if url != nil && *url != "" {
		body += "\n\n🔗 " + *url
	}

	return body
}

func Keyboard(c *model.Competition) dto.Keyboard {
	return dto.Keyboard{
		dto.Row(dto.Button{Label: "➕ В мои", Data: fmt.Sprintf("%s%d", callbackAddPrefix, c.ID)}),
		dto.Row(dto.Button{Label: "← Назад", Data: callbackCatalogList}),
	}
}
