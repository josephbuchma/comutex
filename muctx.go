// This package allows to use functional composition with mutexes.
// loctx
// muctx
package muctx

import (
	"context"
	"sync"
)

// UnlockFn returns true if mutex has been unlocked.
type UnlockFn func() (unlocked bool)

type RWLocker interface {
	sync.Locker
	RLock()
	RUnlock()
}

func IsLocked(ctx context.Context, mu sync.Locker) bool {
	return ctx.Value(mu) != nil
}

// Lock actually locks mutex if given context is not "locked" already, and returns "locked" context.
// If given context is already "locked", it returns it alongside with no-op unlock.
func Lock(ctx context.Context, mu sync.Locker) (lockedCtx context.Context, unlock UnlockFn) {
	if IsLocked(ctx, mu) {
		return ctx, noop
	}
	mu.Lock()
	return withMutex(ctx, mu, mu), wrapUnlock(mu.Unlock)
}

// RLock actually locks RW mutex for reading only if given context is not "locked" already, and returns "locked" context.
// If given context is already "locked", it returns it alongside with no-op unlock.
func RLock(ctx context.Context, mu RWLocker) (lockedCtx context.Context, unlock UnlockFn) {
	if IsLocked(ctx, mu) {
		return ctx, noop
	}
	mu.RLock()
	return withMutex(ctx, mu, mu), wrapUnlock(mu.RUnlock)
}

// Strip removes "unlocked" context without actually unlocking it.
// It must be used when you pass context into new goroutine.
func Strip(ctx context.Context, mu sync.Locker) context.Context {
	return withMutex(ctx, mu, nil)
}

func wrapUnlock(unlock func()) func() bool {
	return func() bool {
		unlock()
		return true
	}
}

func withMutex(ctx context.Context, key sync.Locker, mu interface{}) context.Context {
	return context.WithValue(ctx, key, mu)
}

func noop() bool {
	return false
}
