package pool

import (
	"testing"
)

func TestBytePoolNew(t *testing.T) {
	bp := New(1024)
	if bp.size != 1024 {
		t.Errorf("expected size 1024, got %d", bp.size)
	}
}

func TestBytePoolNewDefaultSize(t *testing.T) {
	bp := New(0)
	if bp.size != 64*1024 {
		t.Errorf("expected default size 64KB, got %d", bp.size)
	}
}

func TestBytePoolGetPut(t *testing.T) {
	bp := New(1024)

	// Get a buffer
	buf := bp.Get()
	if len(buf) != 1024 {
		t.Errorf("expected buffer size 1024, got %d", len(buf))
	}
	if cap(buf) != 1024 {
		t.Errorf("expected buffer capacity 1024, got %d", cap(buf))
	}

	// Write some data
	copy(buf, []byte("test data"))

	// Put it back
	bp.Put(buf)

	// Get another - might be same buffer from pool
	buf2 := bp.Get()
	if len(buf2) != 1024 {
		t.Errorf("expected buffer size 1024 after Put/Get, got %d", len(buf2))
	}
}

func TestBytePoolRejectSmallBuffer(t *testing.T) {
	bp := New(1024)

	// Create a small buffer
	smallBuf := make([]byte, 512)

	// Put should not store it
	bp.Put(smallBuf)

	// Get a buffer - should be a fresh one from pool's New func
	buf := bp.Get()
	if len(buf) != 1024 {
		t.Errorf("expected fresh buffer size 1024, got %d", len(buf))
	}
}

func TestBytePoolAcceptLargeBuffer(t *testing.T) {
	bp := New(1024)

	// Create a larger buffer
	largeBuf := make([]byte, 2048)

	// Put should store it (capacity >= size)
	bp.Put(largeBuf)

	// Get a buffer - should be the large one
	buf := bp.Get()
	if cap(buf) < 2048 {
		t.Errorf("expected large buffer capacity >= 2048, got %d", cap(buf))
	}
}

func TestBytePoolStats(t *testing.T) {
	bp := New(4096)
	stats := bp.Stats()

	if stats["buffer_size"] != 4096 {
		t.Errorf("expected stats buffer_size 4096, got %v", stats["buffer_size"])
	}
}

func TestBytePoolConcurrent(t *testing.T) {
	bp := New(512)
	done := make(chan bool, 10)

	// Concurrent Get/Put operations
	for i := 0; i < 10; i++ {
		go func() {
			buf := bp.Get()
			if len(buf) != 512 {
				t.Errorf("expected size 512, got %d", len(buf))
			}
			copy(buf, []byte("concurrent test"))
			bp.Put(buf)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestBytePoolMultipleSizes(t *testing.T) {
	sizes := []int{512, 1024, 4096, 8192}

	for _, size := range sizes {
		bp := New(size)
		buf := bp.Get()
		if len(buf) != size {
			t.Errorf("size %d: expected buffer size %d, got %d", size, size, len(buf))
		}
	}
}
