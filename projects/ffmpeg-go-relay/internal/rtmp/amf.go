package rtmp

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// AMF0 Markers
const (
	MarkerNumber      = 0x00
	MarkerBoolean     = 0x01
	MarkerString      = 0x02
	MarkerObject      = 0x03
	MarkerNull        = 0x05
	MarkerECMAArray   = 0x08
	MarkerObjectEnd   = 0x09
	MarkerStrictArray = 0x0A
	MarkerDate        = 0x0B
	MarkerLongString  = 0x0C
)

// Limits to prevent DoS attacks
const (
	maxAMFValues    = 1000  // Max number of AMF values in a single decode
	maxAMFStringLen = 65535 // Max string length (AMF0 spec limit)
	maxObjectKeys   = 500   // Max keys in a single object
)

var (
	ErrInvalidMarker   = errors.New("amf: invalid marker")
	ErrEndObject       = errors.New("amf: end of object")
	ErrValueLimit      = errors.New("amf: value limit exceeded")
	ErrStringTooLong   = errors.New("amf: string too long")
	ErrObjectKeyLimit  = errors.New("amf: object key limit exceeded")
)

// DecodeAMF0 decodes a sequence of AMF0 values from the reader
func DecodeAMF0(r io.Reader) ([]interface{}, error) {
	var values []interface{}
	for {
		if len(values) >= maxAMFValues {
			return nil, ErrValueLimit
		}
		v, err := DecodeAMF0Value(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		values = append(values, v)
	}
	return values, nil
}

// DecodeAMF0Value decodes a single AMF0 value
func DecodeAMF0Value(r io.Reader) (interface{}, error) {
	var marker [1]byte
	if _, err := io.ReadFull(r, marker[:]); err != nil {
		return nil, err
	}

	switch marker[0] {
	case MarkerNumber:
		return decodeNumber(r)
	case MarkerBoolean:
		return decodeBoolean(r)
	case MarkerString:
		return decodeString(r)
	case MarkerObject:
		return decodeObject(r)
	case MarkerNull:
		return nil, nil
	case MarkerECMAArray:
		return decodeECMAArray(r)
	case MarkerObjectEnd:
		return nil, ErrEndObject
	default:
		return nil, createInvalidMarkerError(marker[0])
	}
}

func decodeNumber(r io.Reader) (float64, error) {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	bits := binary.BigEndian.Uint64(b[:])
	return math.Float64frombits(bits), nil
}

func decodeBoolean(r io.Reader) (bool, error) {
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return false, err
	}
	return b[0] != 0, nil
}

func decodeString(r io.Reader) (string, error) {
	var lenBuf [2]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return "", err
	}
	length := binary.BigEndian.Uint16(lenBuf[:])

	if length == 0 {
		return "", nil
	}

	if length > maxAMFStringLen {
		return "", ErrStringTooLong
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func decodeObject(r io.Reader) (map[string]interface{}, error) {
	obj := make(map[string]interface{})
	for {
		if len(obj) >= maxObjectKeys {
			return nil, ErrObjectKeyLimit
		}

		key, err := decodeString(r)
		if err != nil {
			return nil, err
		}

		// Empty key can signify end of object in some cases,
		// but usually followed by MarkerObjectEnd (0x09)

		val, err := DecodeAMF0Value(r)
		if err == ErrEndObject {
			break
		}
		if err != nil {
			return nil, err
		}

		obj[key] = val
	}
	return obj, nil
}

func decodeECMAArray(r io.Reader) (map[string]interface{}, error) {
	var countBuf [4]byte
	if _, err := io.ReadFull(r, countBuf[:]); err != nil {
		return nil, err
	}
	// We largely ignore the count in loose parsing and read until ObjectEnd
	return decodeObject(r)
}

func createInvalidMarkerError(marker byte) error {
	return errors.New("amf: unsupported or invalid marker: " + string([]byte{marker}))
}
