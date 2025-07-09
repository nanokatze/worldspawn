package opusfile

import (
	"encoding/binary"
	"strconv"
)

type UnsupportedChannelMappingFamilyError int

func (e UnsupportedChannelMappingFamilyError) Error() string {
	return "opusfile: unsupported channel mapping family %d" + strconv.FormatInt(int64(e), 10)
}

const maxChannelCount = 255

// Ogg Opus bitstream information
type header struct {
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

// TODO: return instead of out param?
func parseHeader(src []byte, header *header) error {
	if len(src) < 19 {
		return ErrBadHeader
	}
	if string(src[0:8]) != "OpusHead" {
		return ErrBadHeader
	}

	header.Version = int(src[8])
	header.ChannelCount = int(src[9])
	header.PreSkip = int(binary.LittleEndian.Uint16(src[10:]))
	header.InputSampleRate = int(binary.LittleEndian.Uint32(src[12:]))
	header.OutputGain = int(int16(binary.LittleEndian.Uint16(src[16:])))
	header.MappingFamily = int(src[18])

	switch header.MappingFamily {
	case 0:
		if header.ChannelCount < 1 || 2 < header.ChannelCount {
			return ErrBadHeader
		}
		header.StreamCount = 1
		header.CoupledCount = header.ChannelCount - 1
		header.Mapping[0] = 0
		header.Mapping[1] = 1

	// TODO: evaluate whether to support channel mapping family 1 for up to 8 channel support

	default:
		return UnsupportedChannelMappingFamilyError(header.MappingFamily)
	}

	return nil
}
