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
)

// fakeChain is a minimal hand-written stand-in for Router's own chain
// dependency — the chain's actual routing logic (what's handled, and in
// what order) is covered by internal/usecase/router/chain's own tests.
// Router's tests only need to prove it resolves the user correctly and
// forwards to (or never reaches) the chain.
type fakeChain struct {
	err    error
	called bool
}

func (c *fakeChain) Handle(context.Context, *model.User, dto.Input) error {
	c.called = true

	return c.err
}

func TestRouter_Route(t *testing.T) {
	t.Parallel()

	const telegramID, userID, chatID int64 = 123, 42, 777

	tests := []struct {
		name     string
		in       dto.Input
		prepare  func(user *Mockuser)
		chain    *fakeChain
		expected func(t assert.TestingT, c *fakeChain, err error)
	}{
		{
			name: "failed to fetch user",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(nil, errors.New("fail"))
			},
			chain: &fakeChain{},
			expected: func(t assert.TestingT, c *fakeChain, err error) {
				assert.Error(t, err)
				assert.False(t, c.called)
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
			chain: &fakeChain{},
			expected: func(t assert.TestingT, c *fakeChain, err error) {
				assert.NoError(t, err)
				assert.False(t, c.called)
			},
		},

		{
			name: "an error from the chain propagates",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID}, nil)
			},
			chain: &fakeChain{err: errors.New("fail")},
			expected: func(t assert.TestingT, c *fakeChain, err error) {
				assert.Error(t, err)
				assert.True(t, c.called)
			},
		},

		{
			name: "nothing in the chain claims it — no-op",
			in:   dto.Input{TelegramID: telegramID, ChatID: chatID, IsStartCmd: true},
			prepare: func(user *Mockuser) {
				user.EXPECT().
					GetByTelegramID(gomock.Any(), telegramID).
					Return(&model.User{ID: userID}, nil)
			},
			chain: &fakeChain{err: nil},
			expected: func(t assert.TestingT, c *fakeChain, err error) {
				assert.NoError(t, err)
				assert.True(t, c.called)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockUser := NewMockuser(ctrl)

			tc.prepare(mockUser)

			r := router.New(tc.chain, mockUser)

			err := r.Route(context.Background(), tc.in)

			tc.expected(t, tc.chain, err)
		})
	}
}
