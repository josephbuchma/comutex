// This package allows to use functional composition with mutexes.
// TODO: rename to loctx
package muctx

import (
	"context"
	"errors"
	"sync"
)

// UnlockFn returns true if mutex has been unlocked.
type UnlockFn func() (unlocked bool)

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
			return ctx, noop, ErrLockUpgrade
		}
		return ctx, noop, nil
	}
	mu.Lock()
	return withMutex(ctx, mu, Locked), wrapUnlock(mu.Unlock), nil
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
		return ctx, noop
	}
	mu.RLock()
	return withMutex(ctx, mu, RLocked), wrapUnlock(mu.RUnlock)
}

// Strip returns "unlocked" context, but does not actually call Unlock on mutex.
// Useful when you are passing locked context to a goroutine.
func Strip(ctx context.Context, mu sync.Locker) context.Context {
	return withMutex(ctx, mu, Unlocked)
}

func wrapUnlock(unlock func()) func() bool {
	return func() bool {
		unlock()
		return true
	}
}

func withMutex(ctx context.Context, key sync.Locker, ls LockStatus) context.Context {
	return context.WithValue(ctx, key, ls)
}

func noop() bool {
	return false
}
