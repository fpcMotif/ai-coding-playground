package pool

import "sync"

// BytePool provides a pool of reusable byte buffers
type BytePool struct {
	pool sync.Pool
	size int
}

// New creates a new byte pool with buffers of given size
func New(size int) *BytePool {
	if size <= 0 {
		size = 64 * 1024 // Default 64KB
	}

	return &BytePool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (bp *BytePool) Get() []byte {
	v := bp.pool.Get()
	buf, ok := v.([]byte)
	if !ok || buf == nil || cap(buf) < bp.size {
		// Fallback: create new buffer if pool returns invalid value
		buf = make([]byte, bp.size)
	}
	return buf[:bp.size] // Ensure full size
}

// Put returns a buffer to the pool
func (bp *BytePool) Put(buf []byte) {
	// Only put back buffers of correct size
	if cap(buf) >= bp.size {
		bp.pool.Put(buf)
	}
}

// Stats returns pool statistics
func (bp *BytePool) Stats() map[string]interface{} {
	return map[string]interface{}{
		"buffer_size": bp.size,
	}
}
