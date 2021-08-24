package muctx_test

import (
	"context"
	"testing"

	"github.com/josephbuchma/muctx"
)

var _ muctx.RWLocker = &mutMock{}

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

func TestLock(t *testing.T) {
	func() {
		mu := &mutMock{}
		ctx := context.Background()
		if muctx.IsLocked(ctx, mu) {
			t.Errorf("Expected fresh mutext to be unlocked")
		}
		assertMutMockEql(t, mutMock{}, mu)
		ctx, unlock := muctx.Lock(ctx, mu)
		defer func() {
			ok := unlock()
			if !ok {
				t.Errorf("Top level mutex unlock should return true")
			}
			assertMutMockEql(t, mutMock{1, 1, 0, 0}, mu)
		}()
		if !muctx.IsLocked(ctx, mu) {
			t.Errorf("Expected mutex to be locked")
		}
		assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)

		func(ctx context.Context) {
			ctx2, unlock2 := muctx.Lock(ctx, mu)
			defer func() {
				ok := unlock2()
				if ok {
					t.Errorf("Top level mutex unlock should return false")
				}
				assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)
			}()
			if !muctx.IsLocked(ctx2, mu) {
				t.Errorf("Expected mutex to be locked")
			}
			assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)

			func(ctx context.Context) {
				ctx3, unlock3 := muctx.Lock(ctx, mu)
				defer func() {
					ok := unlock3()
					if ok {
						t.Errorf("Top level mutex unlock should return false")
					}
					assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)
				}()
				if !muctx.IsLocked(ctx3, mu) {
					t.Errorf("Expected mutex to be locked")
				}
				assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)
			}(ctx2)
		}(ctx)
	}()
}

func TestStrip(t *testing.T) {
	ctx := context.Background()
	mu := &mutMock{}

	ctx, _ = muctx.Lock(ctx, mu)
	assertMutMockEql(t, mutMock{1, 0, 0, 0}, mu)
	sctx := muctx.Strip(ctx, mu)
	if muctx.IsLocked(sctx, mu) {
		t.Error("IsLocked must return false")
	}
	if !muctx.IsLocked(ctx, mu) {
		t.Error("IsLocked must return true")
	}
}
