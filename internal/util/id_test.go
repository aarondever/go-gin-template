package util

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestNewIDIsUUIDv7(t *testing.T) {
	id := NewID()

	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("NewID() = %q, not a valid UUID: %v", id, err)
	}
	if v := parsed.Version(); v != 7 {
		t.Errorf("version = %d, want 7", v)
	}
	if variant := parsed.Variant(); variant != uuid.RFC4122 {
		t.Errorf("variant = %v, want %v", variant, uuid.RFC4122)
	}
}

func TestNewIDFormat(t *testing.T) {
	id := NewID()

	if len(id) != 36 {
		t.Errorf("len(NewID()) = %d, want 36", len(id))
	}
	// Canonical 8-4-4-4-12 grouping.
	groups := strings.Split(id, "-")
	wantLens := []int{8, 4, 4, 4, 12}
	if len(groups) != len(wantLens) {
		t.Fatalf("NewID() = %q, want 5 hyphen-separated groups", id)
	}
	for i, want := range wantLens {
		if len(groups[i]) != want {
			t.Errorf("group %d of %q has length %d, want %d", i, id, len(groups[i]), want)
		}
	}
	if id != strings.ToLower(id) {
		t.Errorf("NewID() = %q, want lower-case hex", id)
	}
}

func TestNewIDIsUnique(t *testing.T) {
	const n = 1000

	seen := make(map[string]bool, n)
	for range n {
		id := NewID()
		if seen[id] {
			t.Fatalf("NewID() returned the duplicate %q", id)
		}
		seen[id] = true
	}
}

// IDs double as sort keys, which is the whole reason for UUIDv7: the timestamp
// prefix must make lexical order match creation order, including within the same
// millisecond.
func TestNewIDIsLexicallyOrdered(t *testing.T) {
	const n = 1000

	ids := make([]string, n)
	for i := range ids {
		ids[i] = NewID()
	}

	if !slices.IsSorted(ids) {
		for i := 1; i < len(ids); i++ {
			if ids[i-1] >= ids[i] {
				t.Fatalf("IDs %d and %d are out of order: %q then %q",
					i-1, i, ids[i-1], ids[i])
			}
		}
	}
}

func TestNewIDConcurrent(t *testing.T) {
	const goroutines = 50
	const perGoroutine = 100

	var mu sync.Mutex
	seen := make(map[string]bool, goroutines*perGoroutine)

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ids := make([]string, perGoroutine)
			for i := range ids {
				ids[i] = NewID()
			}

			mu.Lock()
			defer mu.Unlock()
			for _, id := range ids {
				if seen[id] {
					t.Errorf("NewID() returned the duplicate %q", id)
				}
				seen[id] = true
			}
		}()
	}
	wg.Wait()

	if len(seen) != goroutines*perGoroutine {
		t.Errorf("got %d distinct IDs, want %d", len(seen), goroutines*perGoroutine)
	}
}

func BenchmarkNewID(b *testing.B) {
	for b.Loop() {
		_ = NewID()
	}
}
