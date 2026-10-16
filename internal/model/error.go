package model

import "errors"

var ErrNotFound = errors.New("not found")

// ErrDuplicateBeltPromotion is returned when a belt has already been
// recorded for a user (see belt_promotion's unique constraint).
var ErrDuplicateBeltPromotion = errors.New("belt promotion already recorded")

// ErrUserBlockedBot is returned by the gateway when Telegram reports that a
// user has blocked the bot (API error code 403) — callers that send
// unsolicited messages (e.g. internal/usecase/reminder) use this to disable
// whatever triggered the send instead of retrying.
var ErrUserBlockedBot = errors.New("user blocked the bot")

// ErrSkip is what a router/chain Handler returns when an input isn't its
// case — callers must check for it with errors.Is before treating a
// non-nil return as a real failure. It lives here (not in the chain
// package that defines Handler) purely so every chain sub-package
// (router/chain, router/chain/training, router/chain/schedule) can share
// one sentinel value without importing each other — each declares its own
// private Handler interface instead (see those packages), but errors.Is
// needs this one identity in common.
var ErrSkip = errors.New("skip")
