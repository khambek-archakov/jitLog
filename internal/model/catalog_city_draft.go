package model

import "time"

// CatalogCityDraft marks that the next free-text message from this user
// should be read as a one-off catalog city filter — see CatalogViewFilter
// for where that choice actually ends up stored.
type CatalogCityDraft struct {
	UserID    int64
	CreatedAt time.Time
}
