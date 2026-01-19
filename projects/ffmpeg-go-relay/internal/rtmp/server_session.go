package rtmp

import (
	"bytes"
	"fmt"
	"io"
)

// ServerSession handles the server-side RTMP handshake commands.
type ServerSession struct {
	cs *ChunkStream
	w  io.Writer
}

func NewServerSession(cs *ChunkStream, w io.Writer) *ServerSession {
	return &ServerSession{
		cs: cs,
		w:  w,
	}
}

// Handshake performs the RTMP command handshake up to 'publish'.
// Returns the stream name if successful.
func (s *ServerSession) Handshake() (string, error) {
	// 1. Wait for 'connect'
	cmd, err := s.expectCommand("connect")
	if err != nil {
		return "", fmt.Errorf("wait connect: %w", err)
	}

	// Extract transaction ID
	tid, _ := cmd[1].(float64)

	// Send Window Ack Size (2.5MB)
	if err := s.writeProtocolControl(TypeWindowAck, 2500000); err != nil {
		return "", err
	}
	// Send Set Peer Bandwidth (2.5MB, Dynamic)
	if err := s.writeProtocolControl(TypeSetPeerBW, 2500000, 2); err != nil {
		return "", err
	}
	// Send Set Chunk Size (4096)
	if err := s.writeProtocolControl(TypeSetChunkSize, 4096); err != nil {
		return "", err
	}

	// Send _result for connect
	// Props: fmsVer, capabilities
	// Info: level, code, description, objectEncoding
	props := map[string]interface{}{
		"fmsVer":       "FMS/3,0,1,123",
		"capabilities": 31,
	}
	info := map[string]interface{}{
		"level":          "status",
		"code":           "NetConnection.Connect.Success",
		"description":    "Connection succeeded.",
		"objectEncoding": 0,
	}
	if err := s.writeCommand("_result", tid, props, info); err != nil {
		return "", err
	}

	// 2. Loop until 'publish'
	// Clients may send releaseStream, FCPublish, createStream in various orders.
	streamName := ""

	for {
		msg, err := s.cs.ReadMessage()
		if err != nil {
			return "", err
		}

		if msg.Header.TypeID != TypeAMF0Command && msg.Header.TypeID != TypeAMF20Command {
			continue // Ignore non-commands
		}

		payload := msg.Payload
		if msg.Header.TypeID == TypeAMF20Command {
			if len(payload) == 0 {
				return "", fmt.Errorf("empty AMF3 payload")
			}
			if payload[0] != 0 {
				return "", fmt.Errorf("unsupported AMF3 payload")
			}
			payload = payload[1:]
		}

		vals, err := DecodeAMF0(bytes.NewReader(payload))
		if err != nil {
			return "", err
		}
		if len(vals) < 1 {
			continue
		}

		name, _ := vals[0].(string)
		tid, _ := vals[1].(float64)

		switch name {
		case "releaseStream":
			// ignore, send nothing or result? usually ignore
		case "FCPublish":
			// ignore
		case "createStream":
			// Send _result with StreamID 1
			if err := s.writeCommand("_result", tid, nil, 1); err != nil {
				return "", err
			}
		case "publish":
			if len(vals) >= 4 {
				streamName, _ = vals[3].(string)
			}
			// Send onStatus
			status := map[string]interface{}{
				"level":       "status",
				"code":        "NetStream.Publish.Start",
				"description": "Start publishing",
			}
			// onStatus transaction ID is usually 0
			if err := s.writeCommand("onStatus", 0, nil, status); err != nil {
				return "", err
			}
			return streamName, nil
		}
	}
}

func (s *ServerSession) expectCommand(name string) ([]interface{}, error) {
	for {
		msg, err := s.cs.ReadMessage()
		if err != nil {
			return nil, err
		}
		if msg.Header.TypeID == TypeAMF0Command || msg.Header.TypeID == TypeAMF20Command {
			payload := msg.Payload
			if msg.Header.TypeID == TypeAMF20Command {
				if len(payload) == 0 {
					return nil, fmt.Errorf("empty AMF3 payload")
				}
				if payload[0] != 0 {
					return nil, fmt.Errorf("unsupported AMF3 payload")
				}
				payload = payload[1:]
			}

			vals, err := DecodeAMF0(bytes.NewReader(payload))
			if err != nil {
				return nil, err
			}
			if len(vals) > 0 {
				if n, ok := vals[0].(string); ok && n == name {
					return vals, nil
				}
			}
		}
	}
}

func (s *ServerSession) writeCommand(name string, tid float64, args ...interface{}) error {
	buf := new(bytes.Buffer)
	EncodeAMF0(buf, name, tid)
	EncodeAMF0(buf, args...)

	return s.sendMessage(TypeAMF0Command, buf.Bytes())
}

func (s *ServerSession) writeProtocolControl(typeID uint8, val uint32, extra ...byte) error {
	buf := make([]byte, 4+len(extra))
	// Big Endian
	buf[0] = byte(val >> 24)
	buf[1] = byte(val >> 16)
	buf[2] = byte(val >> 8)
	buf[3] = byte(val)
	copy(buf[4:], extra)

	return s.sendMessage(typeID, buf)
}

func (s *ServerSession) sendMessage(typeID uint8, payload []byte) error {
	// Simple Chunk Writer (Fmt 0, CSID 3 for commands)
	// Chunk Size is assumed 128 (default) unless we changed it.
	// But since we are the server, we use 128 for sending unless we sent SetChunkSize.
	// We sent SetChunkSize=4096 earlier, so we can use larger chunks.
	chunkSize := 4096

	header := make([]byte, 12)
	// Fmt 0, CSID 3 (Command) -> 00 000011 -> 0x03
	// Or CSID 2 for Protocol Control? Usually 2.

	csid := 3
	if typeID < 17 { // Protocol Control
		csid = 2
	}

	header[0] = byte(csid & 0x3f) // Fmt 0

	// Timestamp 0 (3 bytes)
	header[1] = 0
	header[2] = 0
	header[3] = 0

	// Length (3 bytes)
	l := len(payload)
	header[4] = byte(l >> 16)
	header[5] = byte(l >> 8)
	header[6] = byte(l)

	// TypeID
	header[7] = typeID

	// StreamID (4 bytes LE) -> 0
	header[8] = 0
	header[9] = 0
	header[10] = 0
	header[11] = 0

	// Write Header
	if _, err := s.w.Write(header); err != nil {
		return err
	}

	// Write Payload (Chunking if needed)
	written := 0
	for written < l {
		end := written + chunkSize
		if end > l {
			end = l
		}

		if written > 0 {
			// Write continuation header (Fmt 3, CSID)
			h := byte(0xC0 | byte(csid)) // Fmt 3 = 11xxxxxx
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
