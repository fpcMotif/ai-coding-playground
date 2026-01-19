package rtmp

import "fmt"

// FLV Tag Constants
const (
	// Video Frame Types
	FrameKeyframe        = 1
	FrameInterframe      = 2
	FrameDisposableInter = 3
	FrameGeneratedKey    = 4
	FrameInfoCommand     = 5

	// Video Codec IDs
	VideoJPEG    = 1
	VideoSorenson = 2
	VideoScreen   = 3
	VideoOn2VP6   = 4
	VideoOn2VP6Alpha = 5
	VideoScreenV2 = 6
	VideoAVC      = 7 // H.264
	VideoHEVC     = 12 // H.265 (Enhanced RTMP)

	// AVC Packet Types
	AVCPacketSequenceHeader = 0
	AVCPacketNALU           = 1
	AVCPacketEOS            = 2

	// Audio Formats
	AudioLinearPCMPlatform = 0
	AudioADPCM            = 1
	AudioMP3              = 2
	AudioLinearPCMLittle  = 3
	AudioNellymoser16k    = 4
	AudioNellymoser8k     = 5
	AudioNellymoser       = 6
	AudioAAC              = 10
	AudioSpeex            = 11
	AudioMP38k            = 14
)

// VideoHeader represents the parsed FLV Video Tag Header
type VideoHeader struct {
	FrameType       uint8
	CodecID         uint8
	AVCPacketType   uint8 // Only if CodecID == VideoAVC
	CompositionTime int32 // Only if CodecID == VideoAVC
}

// AudioHeader represents the parsed FLV Audio Tag Header
type AudioHeader struct {
	Format      uint8
	SampleRate  int
	SampleSize  uint8
	Stereo      bool
	AACPacketType uint8 // Only if Format == AudioAAC
}

// ParseVideoHeader parses the first 1-5 bytes of a video payload
func ParseVideoHeader(payload []byte) (*VideoHeader, error) {
	if len(payload) < 1 {
		return nil, fmt.Errorf("empty video payload")
	}

	b := payload[0]
	frameType := (b >> 4) & 0x0F
	codecID := b & 0x0F

	h := &VideoHeader{
		FrameType: frameType,
		CodecID:   codecID,
	}

	if codecID == VideoAVC {
		if len(payload) < 2 {
			return nil, fmt.Errorf("short avc payload")
		}
		h.AVCPacketType = payload[1]
		
		if len(payload) >= 5 {
			// Composition Time (CTS) is 24-bit big endian
			cts := int32(uint32(payload[2])<<16 | uint32(payload[3])<<8 | uint32(payload[4]))
			// Sign extension for 24-bit int
			if cts&0x800000 != 0 {
				cts |= ^0xFFFFFF
			}
			h.CompositionTime = cts
		}
	}

	return h, nil
}

// ParseAudioHeader parses the first 1-2 bytes of an audio payload
func ParseAudioHeader(payload []byte) (*AudioHeader, error) {
	if len(payload) < 1 {
		return nil, fmt.Errorf("empty audio payload")
	}

	b := payload[0]
	format := (b >> 4) & 0x0F
	rateIdx := (b >> 2) & 0x03
	sizeIdx := (b >> 1) & 0x01
	typeIdx := b & 0x01

	rates := []int{5500, 11000, 22000, 44100}

	h := &AudioHeader{
		Format:     format,
		SampleRate: rates[rateIdx],
		SampleSize: 8,
		Stereo:     typeIdx == 1,
	}
	if sizeIdx == 1 {
		h.SampleSize = 16
	}

	if format == AudioAAC {
		if len(payload) < 2 {
			return nil, fmt.Errorf("short aac payload")
		}
		h.AACPacketType = payload[1]
	}

	return h, nil
}

func (msg *Message) IsVideoKeyframe() bool {
	if msg.Header.TypeID != TypeVideo {
		return false
	}
	h, err := ParseVideoHeader(msg.Payload)
	if err != nil {
		return false
	}
	return h.FrameType == FrameKeyframe
}

func (msg *Message) IsAVCSequenceHeader() bool {
	if msg.Header.TypeID != TypeVideo {
		return false
	}
	h, err := ParseVideoHeader(msg.Payload)
	if err != nil {
		return false
	}
	return h.CodecID == VideoAVC && h.AVCPacketType == AVCPacketSequenceHeader
}

func (msg *Message) IsAACSequenceHeader() bool {
	if msg.Header.TypeID != TypeAudio {
		return false
	}
	h, err := ParseAudioHeader(msg.Payload)
	if err != nil {
		return false
	}
	return h.Format == AudioAAC && h.AACPacketType == 0 // 0 = Sequence Header
}
