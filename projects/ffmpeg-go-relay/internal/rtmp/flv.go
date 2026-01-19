package rtmp

import (
	"encoding/binary"
	"io"
)

// FLV Tag Types
const (
	TagTypeAudio = 8
	TagTypeVideo = 9
	TagTypeScript = 18
)

// WriteFLVHeader writes the FLV file header
func WriteFLVHeader(w io.Writer, hasAudio, hasVideo bool) error {
	// Signature 'FLV'
	// Version 1
	// Flags (Audio=4, Video=1)
	// HeaderSize 9
	
	flags := uint8(0)
	if hasAudio {
		flags |= 0x04
	}
	if hasVideo {
		flags |= 0x01
	}

	header := []byte{
		'F', 'L', 'V',
		0x01,
		flags,
		0x00, 0x00, 0x00, 0x09,
		0x00, 0x00, 0x00, 0x00, // PreviousTagSize 0
	}

	_, err := w.Write(header)
	return err
}

// MessageToFLVTag writes an RTMP message as an FLV tag to the writer
func MessageToFLVTag(w io.Writer, msg *Message) error {
	// Only Audio, Video, and Scripts (AMF0/AMF3) are valid FLV tags
	// Protocol control messages (ChunkSize, Ack, etc.) must NOT be written to FLV
	
	tagType := msg.Header.TypeID
	if tagType == TypeAMF0Command || tagType == TypeAMF20Command {
		tagType = TagTypeScript
	}

	if tagType != TagTypeAudio && tagType != TagTypeVideo && tagType != TagTypeScript {
		return nil // Skip non-media messages
	}

	dataSize := len(msg.Payload)
	timestamp := msg.Header.Timestamp

	// Tag Header (11 bytes)
	// Type (1)
	// DataSize (3)
	// Timestamp (3)
	// TimestampExtended (1)
	// StreamID (3) - Always 0 in FLV files
	
	buf := make([]byte, 11)
	buf[0] = tagType
	buf[1] = byte(dataSize >> 16)
	buf[2] = byte(dataSize >> 8)
	buf[3] = byte(dataSize)
	
	buf[4] = byte(timestamp >> 16)
	buf[5] = byte(timestamp >> 8)
	buf[6] = byte(timestamp)
	buf[7] = byte(timestamp >> 24) // Extended byte comes at the end of the 3-byte timestamp
	
	buf[8] = 0
	buf[9] = 0
	buf[10] = 0

	// Write Header
	if _, err := w.Write(buf); err != nil {
		return err
	}

	// Write Data
	if _, err := w.Write(msg.Payload); err != nil {
		return err
	}

	// Write PreviousTagSize (4 bytes) = DataSize + 11
	tagSize := uint32(dataSize + 11)
	var sizeBuf [4]byte
	binary.BigEndian.PutUint32(sizeBuf[:], tagSize)
	if _, err := w.Write(sizeBuf[:]); err != nil {
		return err
	}

	return nil
}
