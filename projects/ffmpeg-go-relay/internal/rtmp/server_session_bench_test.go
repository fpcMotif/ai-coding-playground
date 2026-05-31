package rtmp

import (
	"io"
	"testing"
	"time"
)

// A writer that simulates a network connection with syscall overhead
type sysCallWriter struct {
	writes int
}

func (d *sysCallWriter) Write(p []byte) (n int, err error) {
	d.writes++
	// Simulate the fixed overhead of a sys call (~500ns)
	time.Sleep(500 * time.Nanosecond)
	return len(p), nil
}

// simulate unoptimized
type unoptimizedSession struct {
	cs *ChunkStream
	w  io.Writer
}

func (s *unoptimizedSession) sendMessage(typeID uint8, payload []byte) error {
	chunkSize := 4096
	header := make([]byte, 12)
	csid := 3
	if typeID < 17 {
		csid = 2
	}
	header[0] = byte(csid & 0x3f)
	l := len(payload)
	header[4] = byte(l >> 16)
	header[5] = byte(l >> 8)
	header[6] = byte(l)
	header[7] = typeID

	if _, err := s.w.Write(header); err != nil {
		return err
	}
	written := 0
	for written < l {
		end := written + chunkSize
		if end > l {
			end = l
		}
		if written > 0 {
			h := byte(0xC0 | byte(csid))
			if _, err := s.w.Write([]byte{h}); err != nil {
				return err
			}
		}
		if _, err := s.w.Write(payload[written:end]); err != nil {
			return err
		}
		written = end
	}
	return nil
}

func BenchmarkSendMessageExtraLargePayloadUnoptimized(b *testing.B) {
	w := &sysCallWriter{}
	session := &unoptimizedSession{nil, w}
	payload := make([]byte, 100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.sendMessage(TypeAMF0Command, payload)
	}
}

func BenchmarkSendMessageExtraLargePayloadBufio(b *testing.B) {
	w := &sysCallWriter{}
	session := NewServerSession(nil, w)
	payload := make([]byte, 100000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		session.sendMessage(TypeAMF0Command, payload)
	}
}
