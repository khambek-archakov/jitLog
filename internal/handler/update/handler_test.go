package update_test

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/khambek-archakov/jitLog/internal/handler/update"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// fakeUseCase is a minimal hand-written update.UseCase — these tests only
// need to prove Handle's own delivery-loop behavior (dispatch, skip empty
// updates, survive a panic), not real routing logic.
type fakeUseCase struct {
	mu      sync.Mutex
	calls   []dto.Input
	lastCtx context.Context
	fn      func(in dto.Input) error
}

func (f *fakeUseCase) Route(ctx context.Context, in dto.Input) error {
	f.mu.Lock()
	f.calls = append(f.calls, in)
	f.lastCtx = ctx
	f.mu.Unlock()

	if f.fn != nil {
		return f.fn(in)
	}

	return nil
}

func (f *fakeUseCase) deadline() (time.Time, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.lastCtx == nil {
		return time.Time{}, false
	}

	return f.lastCtx.Deadline()
}

func (f *fakeUseCase) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.calls)
}

func messageUpdate(text string) tgbotapi.Update {
	return messageUpdateFrom(42, text)
}

func messageUpdateFrom(telegramID int64, text string) tgbotapi.Update {
	return tgbotapi.Update{
		Message: &tgbotapi.Message{
			MessageID: 1,
			From:      &tgbotapi.User{ID: telegramID},
			Chat:      &tgbotapi.Chat{ID: telegramID},
			Text:      text,
		},
	}
}

func TestHandler_Handle(t *testing.T) {
	t.Parallel()

	t.Run("each update gets a bounded deadline, so a hung call can't hold its slot forever", func(t *testing.T) {
		t.Parallel()

		uc := &fakeUseCase{}
		h := update.New(uc, slog.Default())

		updates := make(chan tgbotapi.Update, 1)
		updates <- messageUpdate("hi")

		ctx, cancel := context.WithCancel(context.Background())
		go h.Handle(ctx, updates)

		require.Eventually(t, func() bool {
			_, ok := uc.deadline()
			return ok
		}, time.Second, time.Millisecond)

		deadline, _ := uc.deadline()
		assert.WithinDuration(t, time.Now().Add(35*time.Second), deadline, 5*time.Second)

		cancel()
	})

	t.Run("dispatches a message update to the use case", func(t *testing.T) {
		t.Parallel()

		uc := &fakeUseCase{}
		h := update.New(uc, slog.Default())

		updates := make(chan tgbotapi.Update, 1)
		updates <- messageUpdate("hi")

		ctx, cancel := context.WithCancel(context.Background())
		go h.Handle(ctx, updates)

		require.Eventually(t, func() bool { return uc.callCount() == 1 }, time.Second, time.Millisecond)

		cancel()
	})

	t.Run("an empty update (neither message nor callback) is never routed", func(t *testing.T) {
		t.Parallel()

		uc := &fakeUseCase{}
		h := update.New(uc, slog.Default())

		updates := make(chan tgbotapi.Update, 2)
		updates <- tgbotapi.Update{}
		updates <- messageUpdate("hi")

		ctx, cancel := context.WithCancel(context.Background())
		go h.Handle(ctx, updates)

		require.Eventually(t, func() bool { return uc.callCount() == 1 }, time.Second, time.Millisecond)

		cancel()
	})

	t.Run("a callback whose originating message Telegram omitted doesn't crash the handler", func(t *testing.T) {
		t.Parallel()

		uc := &fakeUseCase{}
		h := update.New(uc, slog.Default())

		updates := make(chan tgbotapi.Update, 1)
		updates <- tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{ID: "cb-1", From: &tgbotapi.User{ID: 42}, Message: nil},
		}

		ctx, cancel := context.WithCancel(context.Background())

		assert.NotPanics(t, func() {
			go h.Handle(ctx, updates)
			require.Eventually(t, func() bool { return uc.callCount() == 1 }, time.Second, time.Millisecond)
		})

		cancel()
	})

	t.Run("a panic while routing one update doesn't stop the handler from processing the next", func(t *testing.T) {
		t.Parallel()

		var calls int32

		uc := &fakeUseCase{fn: func(dto.Input) error {
			n := atomic.AddInt32(&calls, 1)
			if n == 1 {
				panic("boom")
			}

			return nil
		}}

		h := update.New(uc, slog.Default())

		updates := make(chan tgbotapi.Update, 2)
		updates <- messageUpdate("first, panics")
		updates <- messageUpdate("second, should still run")

		ctx, cancel := context.WithCancel(context.Background())
		go h.Handle(ctx, updates)

		require.Eventually(t, func() bool { return atomic.LoadInt32(&calls) == 2 }, time.Second, time.Millisecond)

		cancel()
	})

	t.Run("an error from the use case is logged, not panicked on", func(t *testing.T) {
		t.Parallel()

		uc := &fakeUseCase{fn: func(dto.Input) error { return errors.New("fail") }}
		h := update.New(uc, slog.Default())

		updates := make(chan tgbotapi.Update, 1)
		updates <- messageUpdate("hi")

		ctx, cancel := context.WithCancel(context.Background())

		assert.NotPanics(t, func() {
			go h.Handle(ctx, updates)
			require.Eventually(t, func() bool { return uc.callCount() == 1 }, time.Second, time.Millisecond)
		})

		cancel()
	})

	t.Run("the same user's own updates are serialized, never running concurrently", func(t *testing.T) {
		t.Parallel()

		var inFlight int32
		var sawOverlap int32

		uc := &fakeUseCase{fn: func(dto.Input) error {
			if atomic.AddInt32(&inFlight, 1) > 1 {
				atomic.StoreInt32(&sawOverlap, 1)
			}

			time.Sleep(20 * time.Millisecond)

			atomic.AddInt32(&inFlight, -1)

			return nil
		}}

		h := update.New(uc, slog.Default())

		const n = 5

		updates := make(chan tgbotapi.Update, n)
		for i := 0; i < n; i++ {
			updates <- messageUpdateFrom(42, "hi")
		}

		ctx, cancel := context.WithCancel(context.Background())
		go h.Handle(ctx, updates)

		require.Eventually(t, func() bool { return uc.callCount() == n }, 2*time.Second, time.Millisecond)

		assert.Zero(t, atomic.LoadInt32(&sawOverlap), "this user's own updates should never run at the same time")

		cancel()
	})

	t.Run("different users' updates run concurrently, not serialized against each other", func(t *testing.T) {
		t.Parallel()

		const n = 5

		release := make(chan struct{})
		t.Cleanup(func() { close(release) })

		var started int32

		uc := &fakeUseCase{fn: func(dto.Input) error {
			atomic.AddInt32(&started, 1)
			<-release

			return nil
		}}

		h := update.New(uc, slog.Default())

		updates := make(chan tgbotapi.Update, n)
		for i := int64(0); i < n; i++ {
			updates <- messageUpdateFrom(i+1, "hi")
		}

		ctx, cancel := context.WithCancel(context.Background())
		go h.Handle(ctx, updates)

		// All n belong to different users, so none of them should be stuck
		// waiting on another one's per-user lock — they should all start
		// (and then block on release, proving they're genuinely concurrent,
		// not just quick) well within the timeout.
		require.Eventually(t, func() bool { return atomic.LoadInt32(&started) == n }, time.Second, time.Millisecond)

		cancel()
	})

	t.Run("stops reading once ctx is cancelled", func(t *testing.T) {
		t.Parallel()

		uc := &fakeUseCase{}
		h := update.New(uc, slog.Default())

		updates := make(chan tgbotapi.Update)

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		go func() {
			h.Handle(ctx, updates)
			close(done)
		}()

		cancel()

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("Handle did not return after ctx was cancelled")
		}
	})
}
