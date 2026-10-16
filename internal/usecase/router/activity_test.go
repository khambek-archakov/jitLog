package router

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/khambek-archakov/jitLog/internal/model"
)

func TestShouldTouchActivity(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)

	t.Run("never touched before — always writes", func(t *testing.T) {
		t.Parallel()

		assert.True(t, shouldTouchActivity(&model.User{}, now))
	})

	t.Run("within the throttle window, no decline pending — skipped", func(t *testing.T) {
		t.Parallel()

		u := &model.User{LastActivityAt: now.Add(-30 * time.Second)}

		assert.False(t, shouldTouchActivity(u, now))
	})

	t.Run("past the throttle window — writes", func(t *testing.T) {
		t.Parallel()

		u := &model.User{LastActivityAt: now.Add(-90 * time.Second)}

		assert.True(t, shouldTouchActivity(u, now))
	})

	t.Run("a still-active decline (declined_at >= last_activity_at) bypasses the throttle", func(t *testing.T) {
		t.Parallel()

		lastActivity := now.Add(-30 * time.Second)
		declinedAt := now.Add(-10 * time.Second) // after last activity — decline is still "active"

		u := &model.User{LastActivityAt: lastActivity, WinbackDeclinedAt: &declinedAt}

		// 30s is well inside the 1-minute throttle, but the still-active
		// decline must force a write anyway — this is what lets the very
		// next action lift it (see shouldTouchActivity's own doc comment).
		assert.True(t, shouldTouchActivity(u, now))
	})

	t.Run("an already-lifted decline (declined_at < last_activity_at) follows the normal throttle", func(t *testing.T) {
		t.Parallel()

		declinedAt := now.Add(-time.Hour)
		lastActivity := now.Add(-30 * time.Second) // activity happened after the decline — already lifted

		u := &model.User{LastActivityAt: lastActivity, WinbackDeclinedAt: &declinedAt}

		assert.False(t, shouldTouchActivity(u, now))
	})
}
