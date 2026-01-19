package rtmp

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"time"
)

const (
	versionByte   = 0x03
	handshakeSize = 1536
)

type HandshakeOptions struct {
	Now  func() uint32
	Rand io.Reader
}

// ClientHandshake performs Client side handshake (Simple or Complex)
// Currently defaults to Simple.
func ClientHandshake(rw io.ReadWriter, opts *HandshakeOptions) error {
	return simpleClientHandshake(rw, opts)
}

// ServerHandshake performs Server side handshake (Auto-detects Simple vs Complex)
func ServerHandshake(rw io.ReadWriter, opts *HandshakeOptions) error {
	nowFn, randReader := defaults(opts)

	// Read C0 (Version)
	c0 := []byte{0}
	if err := readAll(rw, c0); err != nil {
		return err
	}
	if c0[0] != versionByte {
		return errors.New("rtmp: invalid client version")
	}

	// Read C1 (1536 bytes)
	c1 := make([]byte, handshakeSize)
	if err := readAll(rw, c1); err != nil {
		return err
	}

	// Detect Complex vs Simple
	// In Simple, bytes 4-8 are zero.
	// In Complex, we try to validate schema 0 or 1 digest.
	
	// Try Scheme 0 (Digest at ~8)
	// Try Scheme 1 (Digest at ~772)
	// For simplicity, if simple handshake validation fails (zeros check), we treat as complex?
	// Actually, many clients send random non-zero bytes for simple handshake too.
	// The robust way is to try validating the digest.

	var scheme int
	var digest []byte
	var ok bool

	// Check for Simple (heuristic: 4-8 are 0) - Only some clients obey this.
	// ffmpeg often sends 0.
	isSimple := c1[4] == 0 && c1[5] == 0 && c1[6] == 0 && c1[7] == 0
	
	if !isSimple {
		// Try Scheme 1 (Digest at 772+)
		scheme = 1
		digest, ok = validateDigest(c1, scheme, GenuineFPKey)
		if !ok {
			// Try Scheme 0 (Digest at 8+)
			scheme = 0
			digest, ok = validateDigest(c1, scheme, GenuineFPKey)
		}
	}

	if ok {
		// Complex Handshake Response
		return complexServerResponse(rw, c1, scheme, digest, nowFn, randReader)
	}

	// Fallback to Simple Handshake Response
	return simpleServerResponse(rw, c1, nowFn, randReader)
}

func validateDigest(packet []byte, scheme int, key []byte) ([]byte, bool) {
	var offset int
	if scheme == 0 {
		offset = getDigestOffset0(packet)
		offset = (offset % 728) + 12
	} else {
		offset = getDigestOffset1(packet)
		offset = (offset % 728) + 776
	}

	if offset+32 > len(packet) {
		return nil, false
	}

	// Calculate expected digest
	digest := calcDigest(packet, key, offset)
	
	// Compare with packet digest
	if bytes.Equal(digest, packet[offset:offset+32]) {
		return digest, true
	}
	return nil, false
}

func complexServerResponse(rw io.ReadWriter, c1 []byte, scheme int, c1Digest []byte, nowFn func() uint32, randReader io.Reader) error {
	// S0 = 0x03
	if err := writeAll(rw, []byte{versionByte}); err != nil {
		return err
	}

	// S1 construction (Complex)
	s1 := make([]byte, handshakeSize)
	// Time
	binary.BigEndian.PutUint32(s1[0:4], nowFn())
	// Version (0x01000504 for FMS)
	copy(s1[4:8], []byte{0x01, 0x00, 0x05, 0x04}) 
	
	// Random filler
	if _, err := io.ReadFull(randReader, s1[8:]); err != nil {
		return err
	}

	// Write digest into S1
	var offset int
	if scheme == 0 {
		offset = getDigestOffset0(s1)
		offset = (offset % 728) + 12
	} else {
		offset = getDigestOffset1(s1)
		offset = (offset % 728) + 776
	}
	
	digestS1 := calcDigest(s1, GenuineFMSKey, offset)
	copy(s1[offset:], digestS1)

	if err := writeAll(rw, s1); err != nil {
		return err
	}

	// S2 construction
	// S2 is computed from C1's digest
	s2 := make([]byte, handshakeSize)
	if _, err := io.ReadFull(randReader, s2); err != nil {
		return err
	}
	
	// Digest of C1 digest
	tempKey := calcHMAC(GenuineFMSKey, c1Digest)
	digestS2 := calcHMAC(tempKey, s2[:len(s2)-32])
	
	// Put digest at the end
	copy(s2[len(s2)-32:], digestS2)

	if err := writeAll(rw, s2); err != nil {
		return err
	}

	// Read C2
	c2 := make([]byte, handshakeSize)
	if err := readAll(rw, c2); err != nil {
		return err
	}

	return nil
}

func simpleServerResponse(rw io.ReadWriter, c1 []byte, nowFn func() uint32, randReader io.Reader) error {
	s1 := make([]byte, handshakeSize)
	binary.BigEndian.PutUint32(s1[0:4], nowFn())
	copy(s1[4:8], []byte{0, 0, 0, 0})
	if _, err := io.ReadFull(randReader, s1[8:]); err != nil {
		return err
	}

	s2 := make([]byte, handshakeSize)
	copy(s2, c1)
	binary.BigEndian.PutUint32(s2[0:4], nowFn())
	copy(s2[4:8], c1[0:4])

	if err := writeAll(rw, []byte{versionByte}); err != nil {
		return err
	}
	if err := writeAll(rw, s1); err != nil {
		return err
	}
	if err := writeAll(rw, s2); err != nil {
		return err
	}

	c2 := make([]byte, handshakeSize)
	if err := readAll(rw, c2); err != nil {
		return err
	}

	return nil
}

func simpleClientHandshake(rw io.ReadWriter, opts *HandshakeOptions) error {
	nowFn, randReader := defaults(opts)

	c1 := make([]byte, handshakeSize)
	binary.BigEndian.PutUint32(c1[0:4], nowFn())
	copy(c1[4:8], []byte{0, 0, 0, 0})
	if _, err := io.ReadFull(randReader, c1[8:]); err != nil {
		return err
	}

	if err := writeAll(rw, []byte{versionByte}); err != nil {
		return err
	}
	if err := writeAll(rw, c1); err != nil {
		return err
	}

	s0 := []byte{0}
	if err := readAll(rw, s0); err != nil {
		return err
	}
	if s0[0] != versionByte {
		return errors.New("rtmp: invalid server version")
	}

	s1 := make([]byte, handshakeSize)
	if err := readAll(rw, s1); err != nil {
		return err
	}
	s2 := make([]byte, handshakeSize)
	if err := readAll(rw, s2); err != nil {
		return err
	}

	c2 := make([]byte, handshakeSize)
	copy(c2, s1)
	if err := writeAll(rw, c2); err != nil {
		return err
	}

	return nil
}

func defaults(opts *HandshakeOptions) (func() uint32, io.Reader) {
	nowFn := func() uint32 { return uint32(time.Now().Unix()) }
	randReader := rand.Reader
	if opts != nil {
		if opts.Now != nil {
			nowFn = opts.Now
		}
		if opts.Rand != nil {
			randReader = opts.Rand
		}
	}
	return nowFn, randReader
}

func readAll(r io.Reader, buf []byte) error {
	_, err := io.ReadFull(r, buf)
	return err
}

func writeAll(w io.Writer, buf []byte) error {
	for len(buf) > 0 {
		n, err := w.Write(buf)
		if err != nil {
			return err
		}
		buf = buf[n:]
	}
	return nil
}
