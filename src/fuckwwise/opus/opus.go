package opus

// #cgo pkg-config: opus
//
// #include <opus/opus_multistream.h>
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

var ErrBadArg = errors.New("one or more invalid/out of range arguments")

var ErrInvalidPacket = errors.New("the compressed data passed is corrupted")

func opusErr(result C.int) error {
	if result < 0 {
		switch result {
		case C.OPUS_BAD_ARG:
			return ErrBadArg
		case C.OPUS_INVALID_PACKET:
			return ErrInvalidPacket
		case C.OPUS_BUFFER_TOO_SMALL,
			C.OPUS_INTERNAL_ERROR,
			C.OPUS_UNIMPLEMENTED,
			C.OPUS_INVALID_STATE,
			C.OPUS_ALLOC_FAIL:
		}
		// should not happen
		panic(fmt.Sprintf("unexpected opus error code %d", result))
	}
	return nil
}

type MSDecoder struct {
	channels int
	d        unsafe.Pointer
}

func NewMSDecoder(Fs, streams, coupledStreams int, channels []uint8) (*MSDecoder, error) {
	decoderSize := int(C.opus_multistream_decoder_get_size(C.int(streams), C.int(coupledStreams)))
	if err := opusErr(C.int(decoderSize)); err != nil {
		return nil, err
	}
	d := unsafe.Pointer(unsafe.SliceData(make([]byte, decoderSize)))
	if err := opusErr(C.opus_multistream_decoder_init(
		(*C.OpusMSDecoder)(d),
		C.int(Fs),
		C.int(len(channels)),
		C.int(streams),
		C.int(coupledStreams),
		(*C.uchar)(unsafe.SliceData(channels)))); err != nil {
		return nil, err
	}
	return &MSDecoder{
		channels: len(channels),
		d:        d,
	}, nil
}

func (d *MSDecoder) Decode(data []byte, pcm []float32, decodeFEC bool) (int, error) {
	// BUG: we should error out when data is too long

	const maxFrameSize = 120 * 48
	frameSize := len(pcm) / d.channels

	decoded := int(C.opus_multistream_decode_float(
		(*C.OpusMSDecoder)(d.d),
		(*C.uchar)(unsafe.SliceData(data)), C.opus_int32(len(data)),
		(*C.float)(unsafe.SliceData(pcm)), C.int(max(frameSize, maxFrameSize)),
		C.int(bool2int(decodeFEC))))
	if err := opusErr(C.int(decoded)); err != nil {
		return 0, err
	}
	return decoded, nil
}

func PacketFrames(data []byte) (int, error) {
	if len(data) < 1 {
		return 0, ErrInvalidPacket
	}
	c := data[0] & 0x3
	switch c {
	case 0:
		return 1, nil
	case 1, 2:
		return 2, nil
	case 3:
		if len(data) < 2 {
			return 0, ErrInvalidPacket
		}
		return int(data[1] & 0x3f), nil
	}
	panic("unreachable")
}

// Table of frame durations in multiples of 0.1 ms keyed by configuration
// number. See RFC6716, section 3. Internal Framing for details.
var packetSamplesPerFrame = [32]int{
	100, 200, 400, 600, // SILK-only NB
	100, 200, 400, 600, // SILK-only MB
	100, 200, 400, 600, // SILK-only WB
	100, 200, // Hybrid SWB
	100, 200, // Hybrid FB
	25, 50, 100, 200, // CELT-only NB
	25, 50, 100, 200, // CELT-only WB
	25, 50, 100, 200, // CELT-only SWB
	25, 50, 100, 200, // CELT-only FB
}

func PacketSamplesPerFrame(data []byte, Fs int) (int, error) {
	if len(data) < 0 {
		return 0, ErrInvalidPacket
	}
	config := data[0] >> 3
	return Fs * packetSamplesPerFrame[config] / 10000, nil
}

func bool2int(x bool) int {
	if x {
		return 1
	} else {
		return 0
	}
}
