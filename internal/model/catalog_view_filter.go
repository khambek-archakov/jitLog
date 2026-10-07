package model

import "time"

// CatalogViewFilter is an explicit, per-user override of the catalog
// browse screen's city filter — deliberately separate from User.City
// (the profile default): its mere existence means "ignore the profile
// default, use City instead" (City itself may be nil, meaning an explicit
// "show every city" choice, which is still a different state from "no
// override at all"). Changing it never touches the user row.
type CatalogViewFilter struct {
	UserID    int64
	City      *string
	UpdatedAt time.Time
}
