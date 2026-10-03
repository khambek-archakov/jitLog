package update

import (
	"context"
	"log/slog"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// maxConcurrentUpdates bounds how many updates Handle will process at once
// — without a bound, a burst of updates could spawn unbounded goroutines.
const maxConcurrentUpdates = 32

// updateTimeout caps how long a single update may occupy its semaphore
// slot. Without this, a hung downstream call (a stuck DB query, a stalled
// Telegram API request) would hold its slot forever — enough of those and
// every slot is permanently gone, regardless of maxConcurrentUpdates'
// value. Matches main.go's own long-poll Timeout (30s) with a little
// headroom.
const updateTimeout = 35 * time.Second

// UseCase is what this handler translates and dispatches updates to — the
// app's router, deciding between onboarding/training/whatever else exists.
type UseCase interface {
	Route(ctx context.Context, in dto.Input) error
}

// Handler consumes raw Telegram updates, translates each into a
// transport-neutral dto.Input and dispatches it to the usecase.
type Handler struct {
	useCase UseCase
	logger  *slog.Logger
	sem     chan struct{}

	mu      sync.Mutex
	perUser map[int64]*sync.Mutex
}

func New(useCase UseCase, logger *slog.Logger) *Handler {
	return &Handler{
		useCase: useCase,
		logger:  logger,
		sem:     make(chan struct{}, maxConcurrentUpdates),
		perUser: make(map[int64]*sync.Mutex),
	}
}

// Handle reads from updates until ctx is done, dispatching each one to its
// own goroutine (bounded by sem) so one slow or hung update — a stuck DB
// query, a slow Telegram API call — can't block every other user's update
// behind it. Different users' updates run fully in parallel; the same
// user's own updates are still serialized (see handle) exactly as the old
// one-at-a-time loop used to guarantee as a side effect.
func (h *Handler) Handle(ctx context.Context, updates tgbotapi.UpdatesChannel) {
	for {
		select {
		case <-ctx.Done():
			return

		case upd := <-updates:
			if upd.Message == nil && upd.CallbackQuery == nil {
				continue
			}

			select {
			case h.sem <- struct{}{}:
				go h.handle(ctx, upd)
			case <-ctx.Done():
				return
			}
		}
	}
}

// handle processes a single update in its own goroutine. The deferred
// recover means a bug anywhere downstream — including one we haven't found
// yet — logs and drops just this one update instead of taking down the
// whole process (Handle runs in its own goroutine off main.go with nothing
// else to catch a panic).
func (h *Handler) handle(ctx context.Context, upd tgbotapi.Update) {
	defer func() {
		<-h.sem

		if r := recover(); r != nil {
			h.logger.Error("panic while handling update", "panic", r)
		}
	}()

	ctx, cancel := context.WithTimeout(ctx, updateTimeout)
	defer cancel()

	// Holding this for the whole call — not just around a hypothetical
	// "critical section" — is deliberate: nothing downstream (draft
	// read-then-write, the create-record-then-delete-draft sequence in
	// training/create and schedule/create, belt-change's two writes in
	// profile/onboarding) takes any DB-level lock of its own, so a second
	// update for this same user arriving mid-flight could otherwise race
	// it: both read the same stale draft state, or both pass a
	// create-draft precondition before either one deletes it, producing a
	// duplicate record from a single double-tap. A single human can't
	// generate enough throughput for serializing just their own updates to
	// matter; it only ever blocks that same person's own very next tap.
	userLock := h.lockFor(telegramID(upd))
	userLock.Lock()
	defer userLock.Unlock()

	if err := h.useCase.Route(ctx, NewInput(upd)); err != nil {
		h.logger.Error("failed to handle update", "error", err)
	}
}

// lockFor returns the mutex serializing telegramID's own updates against
// each other, creating one on first use. Entries are never removed — for
// this project's scale (a bounded, modest user count) that's a handful of
// bytes per user who has ever messaged the bot, not worth the complexity
// of reference-counted cleanup.
func (h *Handler) lockFor(telegramID int64) *sync.Mutex {
	h.mu.Lock()
	defer h.mu.Unlock()

	l, ok := h.perUser[telegramID]
	if !ok {
		l = &sync.Mutex{}
		h.perUser[telegramID] = l
	}

	return l
}
