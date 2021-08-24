// This package allows to use functional composition with mutexes.
// TODO: rename to loctx
package muctx

import (
	"context"
	"errors"
	"sync"
)

// UnlockFn unlocks mutex and returns stripped context if it's top-level.
// If it isn't top-level, it just returns original context.
type UnlockFn func() context.Context

type RWLocker interface {
	sync.Locker
	RLock()
	RUnlock()
}

// ErrLockUpgrade is returned on attempt to Lock()
// RWMutex that is already locked with RLock.
var ErrLockUpgrade = errors.New("mutex is already locked using RLock")

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

// Lock actually locks mutex if given context is not "locked" already, and returns "locked" context.
// If given context is already "locked", it returns it alongside with no-op unlock.
// If mutex is a RWMutex and is already locked with Rlock, it will return ErrLockUpgrade error.
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

// MustLock actually locks mutex if given context is not "locked" already, and returns "locked" context.
// If given context is already "locked", it returns it alongside with no-op unlock.
// If mutex is a RWMutex and is already locked with Rlock, it will panic.
func MustLock(ctx context.Context, mu sync.Locker) (lockedCtx context.Context, unlock UnlockFn) {
	ctx, unlock, err := Lock(ctx, mu)
	if err != nil {
		panic(err)
	}
	return ctx, unlock
}

// RLock actually locks RW mutex for reading only if given context is not "locked" already, and returns "locked" context.
// If given context is already "locked", it returns it alongside with no-op unlock.
func RLock(ctx context.Context, mu RWLocker) (lockedCtx context.Context, unlock UnlockFn) {
	if Status(ctx, mu) > Unlocked {
		return ctx, noop(ctx)
	}
	mu.RLock()
	return withMutex(ctx, mu, RLocked), wrapUnlock(ctx, mu, mu.RUnlock)
}

// WithStatusUnlocked returns "unlocked" context, but does not actually call Unlock on mutex.
// That is, calling Status(ctx) on retuned context will return Unlocked.
// Useful when you are passing locked context to a goroutine.
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
