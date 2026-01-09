package opusfile

import (
	"encoding/binary"
	"strconv"
)

type UnsupportedChannelMappingFamilyError int32

func (e UnsupportedChannelMappingFamilyError) Error() string {
	return "opusfile: unsupported channel mapping family " + strconv.FormatInt(int64(e), 10)
}

const maxChannelCount = 255

// Ogg Opus bitstream information
type opusHead struct {
	Version         int32
	ChannelCount    int32                  // number of channels
	PreSkip         uint32                 // number of samples that should be discarded from the beginning of the stream
	InputSampleRate uint32                 // sample rate of the original input
	OutputGain      int32                  // gain to apply to the decoded output; 10^(OutputGain/5120)
	MappingFamily   int32                  // channel mapping family
	StreamCount     int32                  // no. of Opus streams in each Ogg packet
	CoupledCount    int32                  // no. of coupled streams
	Mapping         [maxChannelCount]uint8 // [ChannelCount]; channel mapping
}

// TODO: return instead of out param or make this a method on opusHead?
func parseHead(src []byte) (*opusHead, error) {
	if len(src) < 19 {
		return nil, ErrBadHeader
	}
	if string(src[0:8]) != "OpusHead" {
		return nil, ErrBadHeader
	}

	var head opusHead
	head.Version = int32(src[8])
	head.ChannelCount = int32(src[9])
	head.PreSkip = uint32(binary.LittleEndian.Uint16(src[10:]))
	head.InputSampleRate = binary.LittleEndian.Uint32(src[12:])
	head.OutputGain = int32(int16(binary.LittleEndian.Uint16(src[16:])))
	head.MappingFamily = int32(src[18])
	// TODO: evaluate whether to support channel mapping family 1 for up to 8 channel support
	switch head.MappingFamily {
	case 0:
		if head.ChannelCount < 1 || 2 < head.ChannelCount {
			return nil, ErrBadHeader
		}
		head.StreamCount = 1
		head.CoupledCount = head.ChannelCount - 1
		head.Mapping[0] = 0
		head.Mapping[1] = 1

	default:
		return nil, UnsupportedChannelMappingFamilyError(head.MappingFamily)
	}

	return &head, nil
}
