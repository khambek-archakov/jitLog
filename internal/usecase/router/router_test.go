package router_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/khambek-archakov/jitLog/internal/model"
	"github.com/khambek-archakov/jitLog/internal/usecase/dto"
	"github.com/khambek-archakov/jitLog/internal/usecase/router"
	"github.com/khambek-archakov/jitLog/internal/usecase/router/chain"
)

// fakeHandler is a minimal hand-written chain.Handler — the chain's actual
// routing logic (what's handled, and in what order) is covered by
// internal/usecase/router/chain's own tests. Router's tests only need to
// prove the loop itself: the first non-ErrSkip outcome wins, an
// error stops the walk, and an unclaimed input is a no-op.
type fakeHandler struct {
	err    error
	called bool
}

func (h *fakeHandler) Handle(context.Context, *model.User, dto.Input) error {
	h.called = true

	return h.err
}

func TestRouter_Route(t *testing.T) {
	t.Parallel()

	const telegramID, userID, chatID int64 = 123, 42, 777

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(user *Mockuser)
		handlers []chain.Handler
		expected func(t assert.TestingT, err error)
	}{
		{
			name: "failed to fetch user",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(nil, errors.New("fail"))
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "no user, not a /start command — ignored, chain never runs",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, HasMessage: true, Text: "hi"},
			prepare: func(user *Mockuser) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(nil, model.ErrNotFound)
			},
			handlers: []chain.Handler{&fakeHandler{err: nil}},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},

		{
			name: "an error from a handler stops the chain and propagates",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID}, nil)
			},
			handlers: []chain.Handler{
				&fakeHandler{err: errors.New("fail")},
				&fakeHandler{err: nil},
			},
			expected: func(t assert.TestingT, err error) {
				assert.Error(t, err)
			},
		},

		{
			name: "no handler claims it — no-op",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID}, nil)
			},
			handlers: []chain.Handler{&fakeHandler{err: chain.ErrSkip}},
			expected: func(t assert.TestingT, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockUser := NewMockuser(ctrl)

			tc.prepare(mockUser)

			r := router.New(tc.handlers, mockUser)

			err := r.Route(context.Background(), tc.in)

			tc.expected(t, err)
		})
	}
}

func TestRouter_Route_StopsAtFirstHandled(t *testing.T) {
	t.Parallel()

	const telegramID, userID, chatID int64 = 123, 42, 777

	ctrl := gomock.NewController(t)

	mockUser := NewMockuser(ctrl)
	mockUser.EXPECT().
		GetByTelegramID(gomock.Any(), telegramID).
		Return(&model.User{ID: userID}, nil)

	skipped := &fakeHandler{err: chain.ErrSkip}
	claims := &fakeHandler{err: nil}
	neverReached := &fakeHandler{err: nil}

	r := router.New([]chain.Handler{skipped, claims, neverReached}, mockUser)

	err := r.Route(context.Background(), dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true})

	assert.NoError(t, err)
	assert.True(t, skipped.called)
	assert.True(t, claims.called)
	assert.False(t, neverReached.called)
}
