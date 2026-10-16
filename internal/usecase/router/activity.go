package router

import (
	"context"
	"fmt"
	"time"

	"github.com/khambek-archakov/jitLog/internal/model"
)

// activityThrottle caps how often touchActivity writes last_activity_at
// for the same user — DB writes aren't free, and nothing downstream needs
// sub-minute precision. The one exception: a user whose winback decline is
// still "active" always gets written immediately, since that write is
// exactly what's supposed to lift it — see shouldTouchActivity and
// internal/usecase/winback.
const activityThrottle = time.Minute

// touchActivity updates u's last_activity_at in the database and, so the
// rest of this same request sees it too, in u itself — any later write in
// this request (e.g. winback's own SetWinbackDeclined) must not clobber it
// with a stale in-memory value.
func (r *Router) touchActivity(ctx context.Context, u *model.User) error {
	now := r.now()

	if !shouldTouchActivity(u, now) {
		return nil
	}

	if err := r.user.TouchActivity(ctx, u.ID, now); err != nil {
		return fmt.Errorf("touch user activity: %w", err)
	}

	u.LastActivityAt = now

	return nil
}

// shouldTouchActivity is a pure function so the throttle/decline-lifting
// rule is unit-testable without a repo mock.
func shouldTouchActivity(u *model.User, now time.Time) bool {
	declinedStillActive := u.WinbackDeclinedAt != nil && !u.WinbackDeclinedAt.Before(u.LastActivityAt)
	if declinedStillActive {
		return true
	}

	return now.Sub(u.LastActivityAt) >= activityThrottle
}
