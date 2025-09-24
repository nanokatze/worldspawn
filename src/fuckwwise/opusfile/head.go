package opusfile

import (
	"encoding/binary"
	"strconv"
)

type UnsupportedChannelMappingFamilyError int

func (e UnsupportedChannelMappingFamilyError) Error() string {
	return "opusfile: unsupported channel mapping family " + strconv.FormatInt(int64(e), 10)
}

const maxChannelCount = 255

// Ogg Opus bitstream information
type opusHead struct {
	Version         int
	ChannelCount    int                    // number of channels
	PreSkip         int                    // number of samples that should be discarded from the beginning of the stream
	InputSampleRate int                    // sample rate of the original input
	OutputGain      int                    // gain to apply to the decoded output; 10^(OutputGain/5120)
	MappingFamily   int                    // channel mapping family
	StreamCount     int                    // no. of Opus streams in each Ogg packet
	CoupledCount    int                    // no. of coupled streams
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
	head.Version = int(src[8])
	head.ChannelCount = int(src[9])
	head.PreSkip = int(binary.LittleEndian.Uint16(src[10:]))
	head.InputSampleRate = int(binary.LittleEndian.Uint32(src[12:]))
	head.OutputGain = int(int16(binary.LittleEndian.Uint16(src[16:])))
	head.MappingFamily = int(src[18])
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
