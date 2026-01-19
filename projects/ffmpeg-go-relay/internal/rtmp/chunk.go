package rtmp

import (
	"encoding/binary"
	"io"
)

// Chunk Stream Constants
const (
	TypeSetChunkSize = 1
	TypeAbortMessage = 2
	TypeAck          = 3
	TypeWindowAck    = 5
	TypeSetPeerBW    = 6

	TypeAudio = 8
	TypeVideo = 9

	TypeAMF20Command = 17
	TypeAMF0Command  = 20
)

const DefaultChunkSize = 128

type ChunkStream struct {
	r           io.Reader
	rxChunkSize uint32 // Chunk size for receiving (peer sends this)
	txChunkSize uint32 // Chunk size for sending (we send this)
	streams     map[uint32]*StreamState
}

type StreamState struct {
	LastHeader ChunkHeader
	Partial    *Message // Currently assembling message
}

type ChunkHeader struct {
	Fmt       uint8
	CSID      uint32
	Timestamp uint32
	Length    uint32
	TypeID    uint8
	StreamID  uint32

	// Internal tracking
	TimeDelta uint32 // For fmt 1/2
}

type Message struct {
	Header  ChunkHeader
	Payload []byte

	// Internal
	bytesRead uint32
}

func NewChunkStream(r io.Reader) *ChunkStream {
	return &ChunkStream{
		r:           r,
		rxChunkSize: DefaultChunkSize,
		txChunkSize: DefaultChunkSize,
		streams:     make(map[uint32]*StreamState),
	}
}

// ReadMessage reads the next full message from the stream.
// It handles interleaving and protocol control messages automatically.
func (c *ChunkStream) ReadMessage() (*Message, error) {
	for {
		// Read one chunk
		msg, err := c.readChunk()
		if err != nil {
			return nil, err
		}

		// If we got a full message
		if msg != nil {
			// Intercept protocol control messages that affect stream state
			if msg.Header.TypeID == TypeSetChunkSize {
				if len(msg.Payload) >= 4 {
					newSize := binary.BigEndian.Uint32(msg.Payload)
					// FFmpeg and others limit this to valid ranges
					if newSize > 0 && newSize < 0x7FFFFFFF {
						c.rxChunkSize = newSize
					}
				}
			}
			return msg, nil
		}
		// If nil, it was a partial chunk, keep reading
	}
}

// readChunk reads one chunk from the wire and updates state.
// Returns a Message if one was completed, or nil if more chunks are needed.
func (c *ChunkStream) readChunk() (*Message, error) {
	// 1. Read Basic Header
	h1, err := readByte(c.r)
	if err != nil {
		return nil, err
	}

	fmtID := (h1 >> 6) & 0x03
	csID := uint32(h1 & 0x3f)

	if csID == 0 {
		h2, err := readByte(c.r)
		if err != nil {
			return nil, err
		}
		csID = 64 + uint32(h2)
	} else if csID == 1 {
		b := make([]byte, 2)
		if _, err := io.ReadFull(c.r, b); err != nil {
			return nil, err
		}
		csID = 64 + uint32(b[0]) + uint32(b[1])*256
	}

	// Get stream state
	state, exists := c.streams[csID]
	if !exists {
		state = &StreamState{}
		c.streams[csID] = state
	}

	header := state.LastHeader
	header.Fmt = fmtID
	header.CSID = csID

	// 2. Read Message Header based on Fmt
	if fmtID == 0 {
		buf := make([]byte, 11)
		if _, err := io.ReadFull(c.r, buf); err != nil {
			return nil, err
		}
		header.Timestamp = bigUint24(buf[0:3])
		header.Length = bigUint24(buf[3:6])
		header.TypeID = buf[6]
		header.StreamID = binary.LittleEndian.Uint32(buf[7:11])
		header.TimeDelta = 0 // Absolute timestamp
	} else if fmtID == 1 {
		buf := make([]byte, 7)
		if _, err := io.ReadFull(c.r, buf); err != nil {
			return nil, err
		}
		header.TimeDelta = bigUint24(buf[0:3])
		header.Length = bigUint24(buf[3:6])
		header.TypeID = buf[6]
		header.Timestamp = state.LastHeader.Timestamp + header.TimeDelta
	} else if fmtID == 2 {
		buf := make([]byte, 3)
		if _, err := io.ReadFull(c.r, buf); err != nil {
			return nil, err
		}
		header.TimeDelta = bigUint24(buf[0:3])
		header.Timestamp = state.LastHeader.Timestamp + header.TimeDelta
	} else if fmtID == 3 {
		// No header, continuation
		if !exists && state.Partial == nil {
			// This can happen if we just started reading and the peer assumes we know the state.
			// Ideally we error, but for robustness we might assume defaults if strictly needed.
			// For now, allow it but timestamp might be wrong if not careful.
		}
		if state.Partial != nil {
			// Continuation of same message
			header = state.Partial.Header
		} else {
			// Start of new message with same fields as previous
			header.TimeDelta = state.LastHeader.TimeDelta
			header.Timestamp = state.LastHeader.Timestamp + header.TimeDelta
		}
	}

	// 3. Extended Timestamp
	// Logic: If the timestamp field was 0xFFFFFF, we read 4 bytes.
	// NOTE: This applies to Timestamp (fmt 0) or TimeDelta (fmt 1/2).
	tsField := header.Timestamp
	if fmtID == 1 || fmtID == 2 {
		tsField = header.TimeDelta
	}

	if tsField >= 0xFFFFFF {
		var b [4]byte
		if _, err := io.ReadFull(c.r, b[:]); err != nil {
			return nil, err
		}
		ext := binary.BigEndian.Uint32(b[:])
		if fmtID == 0 {
			header.Timestamp = ext
		} else {
			header.TimeDelta = ext
			header.Timestamp = state.LastHeader.Timestamp + header.TimeDelta
		}
	}

	// Update LastHeader for next time (except for fmt 3 partials which don't update timestamp yet)
	state.LastHeader = header

	// 4. Payload Reading
	var msg *Message
	if state.Partial != nil {
		msg = state.Partial
	} else {
		msg = &Message{
			Header:    header,
			Payload:   make([]byte, header.Length),
			bytesRead: 0,
		}
		state.Partial = msg
	}

	// Calculate how much to read
	remaining := msg.Header.Length - msg.bytesRead
	chunkLimit := c.rxChunkSize
	toRead := remaining
	if toRead > chunkLimit {
		toRead = chunkLimit
	}

	if _, err := io.ReadFull(c.r, msg.Payload[msg.bytesRead:msg.bytesRead+toRead]); err != nil {
		return nil, err
	}
	msg.bytesRead += toRead

	// Check if complete
	if msg.bytesRead >= msg.Header.Length {
		state.Partial = nil // Clear partial
		return msg, nil
	}

	// Not complete, return nil
	return nil, nil
}

func readByte(r io.Reader) (byte, error) {
	var b [1]byte
	_, err := io.ReadFull(r, b[:])
	return b[0], err
}

func bigUint24(b []byte) uint32 {
	return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}
