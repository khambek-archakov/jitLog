package chain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
)

// fakeHandler is a minimal hand-written Handler — used here to test
// sequence's own walking mechanics directly and in isolation, independent
// of any real link. This used to be Router's job to prove (it owned the
// loop); now that Router just calls Chain.Handle once, the walk itself is
// entirely sequence's concern.
type fakeHandler struct {
	err    error
	called bool
}

func (h *fakeHandler) Handle(context.Context, *model.User, dto.Input) error {
	h.called = true

	return h.err
}

func TestSequence_Handle(t *testing.T) {
	t.Parallel()

	t.Run("stops at the first non-skip outcome", func(t *testing.T) {
		t.Parallel()

		skipped := &fakeHandler{err: model.ErrSkip}
		claims := &fakeHandler{err: nil}
		neverReached := &fakeHandler{err: nil}

		s := sequence{skipped, claims, neverReached}

		err := s.Handle(context.Background(), &model.User{}, dto.Input{})

		assert.NoError(t, err)
		assert.True(t, skipped.called)
		assert.True(t, claims.called)
		assert.False(t, neverReached.called)
	})

	t.Run("an error from a member stops the walk and propagates", func(t *testing.T) {
		t.Parallel()

		failing := &fakeHandler{err: errors.New("fail")}
		neverReached := &fakeHandler{err: nil}

		s := sequence{failing, neverReached}

		err := s.Handle(context.Background(), &model.User{}, dto.Input{})

		assert.Error(t, err)
		assert.False(t, neverReached.called)
	})

	t.Run("every member skipping returns model.ErrSkip", func(t *testing.T) {
		t.Parallel()

		s := sequence{&fakeHandler{err: model.ErrSkip}, &fakeHandler{err: model.ErrSkip}}

		err := s.Handle(context.Background(), &model.User{}, dto.Input{})

		assert.ErrorIs(t, err, model.ErrSkip)
	})
}
