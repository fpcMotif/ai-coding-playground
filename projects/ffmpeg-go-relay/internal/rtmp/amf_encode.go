package rtmp

import (
	"encoding/binary"
	"io"
	"sort"
)

func EncodeAMF0(w io.Writer, values ...interface{}) error {
	for _, v := range values {
		if err := encodeValue(w, v); err != nil {
			return err
		}
	}
	return nil
}

func encodeValue(w io.Writer, v interface{}) error {
	switch t := v.(type) {
	case string:
		return encodeString(w, t)
	case float64:
		return encodeNumber(w, t)
	case int:
		return encodeNumber(w, float64(t))
	case bool:
		return encodeBoolean(w, t)
	case map[string]interface{}:
		return encodeObject(w, t)
	case nil:
		return encodeNull(w)
	default:
		return nil // Skip unsupported types or error?
	}
}

func encodeNumber(w io.Writer, n float64) error {
	if _, err := w.Write([]byte{MarkerNumber}); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, n)
}

func encodeBoolean(w io.Writer, b bool) error {
	if _, err := w.Write([]byte{MarkerBoolean}); err != nil {
		return err
	}
	val := byte(0)
	if b {
		val = 1
	}
	return binary.Write(w, binary.BigEndian, val)
}

func encodeString(w io.Writer, s string) error {
	if _, err := w.Write([]byte{MarkerString}); err != nil {
		return err
	}
	length := uint16(len(s))
	if err := binary.Write(w, binary.BigEndian, length); err != nil {
		return err
	}
	_, err := w.Write([]byte(s))
	return err
}

func encodeObject(w io.Writer, m map[string]interface{}) error {
	if _, err := w.Write([]byte{MarkerObject}); err != nil {
		return err
	}
	
	// Sort keys for deterministic output (optional but good)
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		// Write key
		length := uint16(len(k))
		if err := binary.Write(w, binary.BigEndian, length); err != nil {
			return err
		}
		if _, err := w.Write([]byte(k)); err != nil {
			return err
		}
		
		// Write value
		if err := encodeValue(w, m[k]); err != nil {
			return err
		}
	}

	// Write End Marker (00 00 09)
	if _, err := w.Write([]byte{0x00, 0x00, MarkerObjectEnd}); err != nil {
		return err
	}
	return nil
}

func encodeNull(w io.Writer) error {
	_, err := w.Write([]byte{MarkerNull})
	return err
}
