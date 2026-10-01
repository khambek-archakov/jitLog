package model

import "errors"

var ErrNotFound = errors.New("not found")

// ErrDuplicateBeltPromotion is returned when a belt has already been
// recorded for a user (see belt_promotion's unique constraint).
var ErrDuplicateBeltPromotion = errors.New("belt promotion already recorded")
