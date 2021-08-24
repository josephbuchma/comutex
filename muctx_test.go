package muctx_test

import (
	"context"
	"sync"
	"testing"

	"github.com/josephbuchma/muctx"
)

var _ muctx.RWLocker = &mutMock{}

func TestLock(t *testing.T) {
	mu := &mutMock{}
	ctx := context.Background()
	originalCtx := ctx
	if muctx.Status(ctx, mu) > muctx.Unlocked {
		t.Errorf("Expected fresh mutext to be unlocked")
	}
	assertMutMockEql(t, mutMock{}, mu)
	ctx, unlock, err := muctx.Lock(ctx, mu)
	assertNilErr(t, err)
	defer func() {
		unlockedCtx := unlock()
		if isLocked(unlockedCtx, mu) {
			t.Errorf("Top level mutex unlock must unlock")
		}
		if originalCtx != unlockedCtx {
			t.Errorf("unlock should return original context")
		}
		assertMutMockEql(t, mutMock{1, 1, 0, 0}, mu)
	}()
	if muctx.Status(ctx, mu) == muctx.Unlocked {
		t.Errorf("Expected mutex to be locked")
	}
	assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)

	func(ctx context.Context) {
		ctx2, unlock2, err := muctx.Lock(ctx, mu)
		assertNilErr(t, err)
		defer func() {
			if !isLocked(unlock2(), mu) {
				t.Errorf("Nested mutex must remain locked after unlock")
			}
			assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)
		}()
		if muctx.Status(ctx2, mu) == muctx.Unlocked {
			t.Errorf("Expected mutex to be locked")
		}
		assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)

		func(ctx context.Context) {
			ctx3, unlock3, err := muctx.Lock(ctx, mu)
			assertNilErr(t, err)
			defer func() {
				if !isLocked(unlock3(), mu) {
					t.Errorf("Nested mutex must remain locked after unlock")
				}
				assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)
			}()
			if muctx.Status(ctx3, mu) == muctx.Unlocked {
				t.Errorf("Expected mutex to be locked")
			}
			assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)
		}(ctx2)
	}(ctx)
}

func TestErrLockUpgrade(t *testing.T) {
	ctx := context.Background()
	mu := &mutMock{}
	ctx, _ = muctx.RLock(ctx, mu)
	ctx, _, err := muctx.Lock(ctx, mu)
	if err != muctx.ErrLockUpgrade {
		t.Errorf("expected ErrLockUpgrade")
	}
	if muctx.Status(ctx, mu) != muctx.RLocked {
		t.Errorf("expected RLocked status")
	}
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic")
			}
		}()
		muctx.MustLock(ctx, mu)
	}()
}

func TestStrip(t *testing.T) {
	ctx := context.Background()
	mu := &mutMock{}

	ctx, _ = muctx.MustLock(ctx, mu)
	assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)
	sctx := muctx.Strip(ctx, mu)
	if muctx.Status(sctx, mu) != muctx.Unlocked {
		t.Error("status must be unlocked")
	}
	if muctx.Status(ctx, mu) != muctx.Locked {
		t.Error("status must be locked")
	}
}

func isLocked(ctx context.Context, mu sync.Locker) bool {
	return muctx.Status(ctx, mu) > muctx.Unlocked
}

type mutMock struct {
	locked    int
	unlocked  int
	rlocked   int
	runlocked int
}

func (ml *mutMock) Lock() {
	ml.locked++
}

func (ml *mutMock) Unlock() {
	ml.unlocked++
}

func (ml *mutMock) RLock() {
	ml.rlocked++
}
func (ml *mutMock) RUnlock() {
	ml.runlocked++
}

func assertMutMockEql(t *testing.T, expect mutMock, actual *mutMock) {
	if expect != *actual {
		t.Errorf("expected %#v, got %#v", expect, actual)
	}
}

func assertNilErr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Expected nil error")
	}
}
