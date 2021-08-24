// Package comutex: Context + Mutex = Comutex.
// Main purpose of this package is to allow nested calls
// of functions that lock same mutex.
// Bot Mutex and RWMutex are supported.
package comutex

import (
	"context"
	"errors"
	"sync"
)

// UnlockFn unlocks mutex and returns original context if it's top-level,
// otherwise it just returns original context (the one passed to Lock, MustLock or RLock).
type UnlockFn func() context.Context

// RWLocker is compatible with sync.RWMutex
type RWLocker interface {
	sync.Locker
	RLock()
	RUnlock()
}

// ErrLockUpgrade is returned on attempt to Lock() rwmutex
// that is already locked with RLock.
var ErrLockUpgrade = errors.New("can't Lock() - mutex is already locked using RLock")

// LockStatus is either "unlocked", "locked", or "locked with RLock"
type LockStatus uint32

// Mutex lock statuses
const (
	Unlocked LockStatus = iota
	RLocked
	Locked
)

func (ls LockStatus) String() string {
	switch ls {
	case Unlocked:
		return "unlocked"
	case RLocked:
		return "locked with RLock"
	case Locked:
		return "locked"
	}
	panic("invalid lock state")
}

// Status returns lock status in given context.
func Status(ctx context.Context, mu sync.Locker) LockStatus {
	v, _ := ctx.Value(mu).(LockStatus)
	return v
}

// Lock locks given mutex and returns "locked" context with unlock function.
// If given context is already "locked" (e.g. Status(ctx) != Unlocked) one of two things possible:
// - if Status(ctx) is Locked - it returns same context with no-op unlock function.
// - if Status(ctx) is RLocked - same as above with ErrLockUpgrade
func Lock(ctx context.Context, mu sync.Locker) (context.Context, UnlockFn, error) {
	if status := Status(ctx, mu); status > Unlocked {
		if status != Locked {
			return ctx, noop(ctx), ErrLockUpgrade
		}
		return ctx, noop(ctx), nil
	}
	mu.Lock()
	return withMutex(ctx, mu, Locked), wrapUnlock(ctx, mu, mu.Unlock), nil
}

// MustLock is same as Lock, but instead of returning an error it'll panic.
func MustLock(ctx context.Context, mu sync.Locker) (lockedCtx context.Context, unlock UnlockFn) {
	ctx, unlock, err := Lock(ctx, mu)
	if err != nil {
		panic(err)
	}
	return ctx, unlock
}

// RLock locks given mutex with RLock and returns "locked" context with unlock function.
// If given context is already "locked" (e.g. Status(ctx) != Unlocked),
// it returns same context with no-op unlock function (even if mutex was locked with Lock()).
func RLock(ctx context.Context, mu RWLocker) (lockedCtx context.Context, unlock UnlockFn) {
	if Status(ctx, mu) > Unlocked {
		return ctx, noop(ctx)
	}
	mu.RLock()
	return withMutex(ctx, mu, RLocked), wrapUnlock(ctx, mu, mu.RUnlock)
}

// WithStatusUnlocked returns "unlocked" context, but does not actually call Unlock on mutex.
// That is, calling Status(ctx) on retuned context will return Unlocked.
// Useful when you need to pass locked context to a goroutine.
func WithStatusUnlocked(ctx context.Context, mu sync.Locker) context.Context {
	return withMutex(ctx, mu, Unlocked)
}

func wrapUnlock(ctx context.Context, mu sync.Locker, unlock func()) UnlockFn {
	return func() context.Context {
		unlock()
		return ctx
	}
}

func withMutex(ctx context.Context, key sync.Locker, ls LockStatus) context.Context {
	return context.WithValue(ctx, key, ls)
}

func noop(ctx context.Context) UnlockFn {
	return func() context.Context {
		return ctx
	}
}
