package database

import (
	"sync"
	"testing"
	"time"
)

// Test basic Get/Set functionality
func TestMemoryValues_SetAndGet(t *testing.T) {
	mv := NewMemoryValues[string, string](time.Minute)
	defer mv.Close()

	// Set a value
	mv.Set("key1", "value1")

	// Get the value
	val, ok := mv.Get("key1")
	if !ok {
		t.Fatal("Expected to find key1")
	}
	if val != "value1" {
		t.Fatalf("Expected value1, got %s", val)
	}
}

// Test Get with non-existent key
func TestMemoryValues_GetNonExistent(t *testing.T) {
	mv := NewMemoryValues[string, string](time.Minute)
	defer mv.Close()

	// Try to get a non-existent key
	_, ok := mv.Get("nonexistent")
	if ok {
		t.Fatal("Expected to not find nonexistent key")
	}
}

// Test Delete functionality
func TestMemoryValues_Delete(t *testing.T) {
	mv := NewMemoryValues[string, string](time.Minute)
	defer mv.Close()

	// Set and then delete
	mv.Set("key1", "value1")
	mv.Delete("key1")

	// Verify it's deleted
	_, ok := mv.Get("key1")
	if ok {
		t.Fatal("Expected key1 to be deleted")
	}
}

// Test expiration of values
func TestMemoryValues_Expiration(t *testing.T) {
	// Use a short timeout for testing
	mv := NewMemoryValues[string, string](100 * time.Millisecond)
	defer mv.Close()

	// Set a value
	mv.Set("key1", "value1")

	// Immediately get it - should work
	val, ok := mv.Get("key1")
	if !ok || val != "value1" {
		t.Fatal("Expected to find key1 immediately after setting")
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Try to get expired value
	_, ok = mv.Get("key1")
	if ok {
		t.Fatal("Expected key1 to be expired")
	}
}

// Test expiration cleanup by periodic ticker
func TestMemoryValues_ExpirationCleanup(t *testing.T) {
	// Use a short timeout for testing
	mv := NewMemoryValues[string, string](100 * time.Millisecond)
	defer mv.Close()

	// Set multiple values
	mv.Set("key1", "value1")
	mv.Set("key2", "value2")
	mv.Set("key3", "value3")

	// Wait for expiration and cleanup
	time.Sleep(250 * time.Millisecond)

	// Verify all keys are cleaned up
	mv.mu.RLock()
	count := len(mv.values)
	mv.mu.RUnlock()

	if count != 0 {
		t.Fatalf("Expected all keys to be cleaned up, but found %d keys", count)
	}
}

// Test concurrent access (no race detector errors)
func TestMemoryValues_ConcurrentAccess(t *testing.T) {
	mv := NewMemoryValues[int, string](time.Second)
	defer mv.Close()

	var wg sync.WaitGroup
	iterations := 100

	// Concurrent writes
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				mv.Set(id*iterations+j, "value")
			}
		}(i)
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, _ = mv.Get(id*iterations + j)
			}
		}(i)
	}

	// Concurrent deletes
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				mv.Delete(id*iterations + j)
			}
		}(i)
	}

	wg.Wait()
}

// Test DeleteHandler interface
type testDeleteHandler struct {
	deleted bool
	mu      sync.Mutex
}

func (h *testDeleteHandler) OnDelete() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.deleted = true
	return nil
}

func TestMemoryValues_DeleteHandler(t *testing.T) {
	mv := NewMemoryValues[string, *testDeleteHandler](time.Minute)
	defer mv.Close()

	handler := &testDeleteHandler{}
	mv.Set("key1", handler)

	// Delete the key
	mv.Delete("key1")

	// Check if OnDelete was called
	handler.mu.Lock()
	deleted := handler.deleted
	handler.mu.Unlock()

	if !deleted {
		t.Fatal("Expected OnDelete to be called")
	}
}

// Test DeleteHandler on expiration
func TestMemoryValues_DeleteHandlerOnExpiration(t *testing.T) {
	mv := NewMemoryValues[string, *testDeleteHandler](50 * time.Millisecond)
	defer mv.Close()

	handler := &testDeleteHandler{}
	mv.Set("key1", handler)

	// Wait for expiration and cleanup
	time.Sleep(200 * time.Millisecond)

	// Check if OnDelete was called during cleanup
	handler.mu.Lock()
	deleted := handler.deleted
	handler.mu.Unlock()

	if !deleted {
		t.Fatal("Expected OnDelete to be called on expiration")
	}
}

// Test Close stops cleanup goroutine
func TestMemoryValues_Close(t *testing.T) {
	mv := NewMemoryValues[string, string](time.Second)

	// Close the memory values
	mv.Close()

	// Wait a bit to ensure cleanup goroutine has stopped
	time.Sleep(100 * time.Millisecond)

	// We can't directly test if the goroutine stopped,
	// but at least we verify Close doesn't panic
}

// Test zero timeout (no expiration)
func TestMemoryValues_NoExpiration(t *testing.T) {
	mv := NewMemoryValues[string, string](0)
	defer mv.Close()

	// Set a value with no expiration
	mv.Set("key1", "value1")

	// Wait a bit
	time.Sleep(200 * time.Millisecond)

	// Value should still be there
	val, ok := mv.Get("key1")
	if !ok || val != "value1" {
		t.Fatal("Expected key1 to still exist with no expiration")
	}
}

// Benchmark Set operation
func BenchmarkMemoryValues_Set(b *testing.B) {
	mv := NewMemoryValues[int, string](time.Minute)
	defer mv.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mv.Set(i, "value")
	}
}

// Benchmark Get operation
func BenchmarkMemoryValues_Get(b *testing.B) {
	mv := NewMemoryValues[int, string](time.Minute)
	defer mv.Close()

	// Pre-populate with some data
	for i := 0; i < 1000; i++ {
		mv.Set(i, "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mv.Get(i % 1000)
	}
}

// Benchmark concurrent access
func BenchmarkMemoryValues_ConcurrentAccess(b *testing.B) {
	mv := NewMemoryValues[int, string](time.Minute)
	defer mv.Close()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				mv.Set(i, "value")
			} else {
				_, _ = mv.Get(i)
			}
			i++
		}
	})
}
