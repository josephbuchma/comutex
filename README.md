
# comutex [![Go Report Card](https://goreportcard.com/badge/github.com/josephbuchma/comutex)](https://goreportcard.com/badge/github.com/josephbuchma/comutex) 

> Context + Mutex = Comutex

Comutex is a Go package which allows to perform nested calls of functions that lock same mutex 
in the same thread/goroutine with a help of `context.Context`.
Both Mutex and RWMutex are supported.

## Example

```go
package comutex_test

import (
	"context"
	"fmt"
	"sync"

	"github.com/josephbuchma/comutex"
)

// IntKVStore is a simple shared stirng=>int KV store.
// Even though all of its methods are locking mutex,
// Get method is used inside of Sum,
// and Has is used inside of Insert and Update - and it doesn't cause a deadlock.
type IntKVStore struct {
	mu sync.RWMutex
	m  map[string]int
}

func (s *IntKVStore) Get(ctx context.Context, key string) int {
	_, unlock := comutex.RLock(ctx, &s.mu)
	defer unlock()
	return s.m[key]
}

func (s *IntKVStore) Has(ctx context.Context, key string) bool {
	_, unlock := comutex.RLock(ctx, &s.mu)
	defer unlock()
	_, ok := s.m[key]
	return ok
}

func (s *IntKVStore) Insert(ctx context.Context, key string, v int) bool {
	ctx, unlock := comutex.MustLock(ctx, &s.mu)
	defer unlock()
	if s.Has(ctx, key) {
		return false
	}
	s.m[key] = v
	return true
}

func (s *IntKVStore) Update(ctx context.Context, key string, v int) bool {
	ctx, unlock := comutex.MustLock(ctx, &s.mu)
	defer unlock()
	if s.Has(ctx, key) {
		s.m[key] = v
		return true
	}
	return false
}

func (s *IntKVStore) Sum(ctx context.Context, keys ...string) int {
	ctx, unlock := comutex.RLock(ctx, &s.mu)
	defer unlock()
	r := 0
	for _, key := range keys {
		r += s.Get(ctx, key)
	}
	return r
}

func Example() {
	ctx := context.Background()
	s := &IntKVStore{m: map[string]int{}}
	s.Insert(ctx, "a", 1)
	s.Insert(ctx, "b", 2)
	s.Update(ctx, "b", 3)

	fmt.Println("a+b:", s.Sum(ctx, "a", "b"))

	// Output:
	// a+b: 4
}

```


## License

MIT