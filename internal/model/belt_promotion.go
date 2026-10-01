package model

import "time"

// BeltPromotion records a belt as having started on PromotedAt — see
// internal/repository/user's belt_promotion table. Ordered by PromotedAt,
// these let an "as of" lookup answer "what belt was I at on date X."
type BeltPromotion struct {
	ID         int64
	UserID     int64
	Belt       Belt
	PromotedAt time.Time
	CreatedAt  time.Time
}
