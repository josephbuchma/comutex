package comutex_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/josephbuchma/comutex"
)

var _ comutex.RWLocker = &mutMock{}

func TestRLock(t *testing.T) {
	mu := &mutMock{}
	ctx := context.Background()
	originalCtx := ctx
	if comutex.Status(ctx, mu) > comutex.Unlocked {
		t.Errorf("Expected fresh mutext to be unlocked")
	}
	assertMutMockEql(t, mutState{}, mu)
	ctx, unlock := comutex.RLock(ctx, mu)
	defer func() {
		unlockedCtx := unlock()
		if isLocked(unlockedCtx, mu) {
			t.Errorf("Top level mutex unlock must unlock")
		}
		if originalCtx != unlockedCtx {
			t.Errorf("unlock should return original context")
		}
		assertMutMockEql(t, mutState{0, 0, 1, 1}, mu)
	}()
	if comutex.Status(ctx, mu) == comutex.Unlocked {
		t.Errorf("Expected mutex to be locked")
	}
	assertMutMockEql(t, mutState{0, 0, 1, 0}, mu)

	func(ctx context.Context) {
		ctx2, unlock2 := comutex.RLock(ctx, mu)
		defer func() {
			if !isLocked(unlock2(), mu) {
				t.Errorf("Nested mutex must remain locked after unlock")
			}
			assertMutMockEql(t, mutState{0, 0, 1, 0}, mu)
		}()
		if comutex.Status(ctx2, mu) == comutex.Unlocked {
			t.Errorf("Expected mutex to be locked")
		}
		assertMutMockEql(t, mutState{0, 0, 1, 0}, mu)

		func(ctx context.Context) {
			ctx3, unlock3 := comutex.RLock(ctx, mu)
			defer func() {
				if !isLocked(unlock3(), mu) {
					t.Errorf("Nested mutex must remain locked after unlock")
				}
				assertMutMockEql(t, mutState{0, 0, 1, 0}, mu)
			}()
			if comutex.Status(ctx3, mu) == comutex.Unlocked {
				t.Errorf("Expected mutex to be locked")
			}
			assertMutMockEql(t, mutState{0, 0, 1, 0}, mu)
		}(ctx2)
	}(ctx)
}

func assertStatus(t *testing.T, ctx context.Context, mu sync.Locker, status comutex.LockStatus) {
	s := comutex.Status(ctx, mu)
	if s != status {
		t.Errorf("expected lock status %q, got %q", status, status)
	}
}

func TestLock(t *testing.T) {
	mu := &mutMock{}
	ctx := context.Background()
	originalCtx := ctx
	assertStatus(t, ctx, mu, comutex.Unlocked)
	assertMutMockEql(t, mutState{}, mu)
	ctx, unlock, err := comutex.Lock(ctx, mu)
	assertNilErr(t, err)
	defer func() {
		unlockedCtx := unlock()
		assertStatus(t, unlockedCtx, mu, comutex.Unlocked)
		if originalCtx != unlockedCtx {
			t.Errorf("unlock should return original context")
		}
		assertMutMockEql(t, mutState{1, 1, 0, 0}, mu)
	}()
	assertStatus(t, ctx, mu, comutex.Locked)
	assertMutMockEql(t, mutState{1, 0, 0, 0}, mu)

	func(ctx context.Context) {
		ctx2, unlock2, err := comutex.Lock(ctx, mu)
		assertNilErr(t, err)
		defer func() {
			if !isLocked(unlock2(), mu) {
				t.Errorf("Nested mutex must remain locked after unlock")
			}
			assertMutMockEql(t, mutState{1, 0, 0, 0}, mu)
		}()
		assertStatus(t, ctx2, mu, comutex.Locked)
		assertMutMockEql(t, mutState{1, 0, 0, 0}, mu)

		func(ctx context.Context) {
			ctx3, unlock3, err := comutex.Lock(ctx, mu)
			assertNilErr(t, err)
			defer func() {
				if !isLocked(unlock3(), mu) {
					t.Errorf("Nested mutex must remain locked after unlock")
				}
				assertMutMockEql(t, mutState{1, 0, 0, 0}, mu)
			}()
			assertStatus(t, ctx3, mu, comutex.Locked)
			assertMutMockEql(t, mutState{1, 0, 0, 0}, mu)
		}(ctx2)
	}(ctx)
}

func TestErrLockUpgrade(t *testing.T) {
	ctx := context.Background()
	mu := &mutMock{}
	ctx, _ = comutex.RLock(ctx, mu)
	ctx, _, err := comutex.Lock(ctx, mu)
	if err != comutex.ErrLockUpgrade {
		t.Errorf("expected ErrLockUpgrade")
	}
	if comutex.Status(ctx, mu) != comutex.RLocked {
		t.Errorf("expected RLocked status")
	}
	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic")
			}
		}()
		comutex.MustLock(ctx, mu)
	}()
}

func TestWithStatusUnlocked(t *testing.T) {
	ctx := context.Background()
	mu := &mutMock{}

	ctx, _ = comutex.MustLock(ctx, mu)
	if comutex.Status(ctx, mu) != comutex.Locked {
		t.Error("status must be locked")
	}
	assertMutMockEql(t, mutState{1, 0, 0, 0}, mu)
	sctx := comutex.WithStatusUnlocked(ctx, mu)
	if comutex.Status(sctx, mu) != comutex.Unlocked {
		t.Error("status must be unlocked")
	}
	assertMutMockEql(t, mutState{1, 0, 0, 0}, mu)
}

func TestLockStatus(t *testing.T) {
	for _, status := range []comutex.LockStatus{0, 1, 2} {
		if fmt.Sprintf("%d", status) == status.String() {
			t.Errorf("missing string for status %d", status)
		}
	}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("calling String on invalid lock status did not panic")
		}
	}()
	_ = comutex.LockStatus(3).String()
}

func isLocked(ctx context.Context, mu sync.Locker) bool {
	return comutex.Status(ctx, mu) > comutex.Unlocked
}

type mutState struct {
	locked    int
	unlocked  int
	rlocked   int
	runlocked int
}

type mutMock struct {
	mutState
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

func assertMutMockEql(t *testing.T, expect mutState, actual *mutMock) {
	if expect != actual.mutState {
		t.Errorf("expected %#v, got %#v", expect, actual)
	}
}

func assertNilErr(t *testing.T, err error) {
	if err != nil {
		t.Errorf("Expected nil error")
	}
}
