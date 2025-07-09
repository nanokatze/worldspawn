// Package opusfile provides functions for reading and decoding Opus encapsulated in
// Ogg. See RFC7845 for details.
package opusfile

// TODO: clean up ogg code

import (
	"errors"
	"io"
	"math"
	"unsafe"

	"worldspawn/fuckwwise/opus"
)

var ErrBadHeader = errors.New("invalid Ogg Opus header")

var ErrBadPacket = errors.New("packet failed to decode properly")

type tags struct {
}

func parseTags(src []byte, tags *tags) error {
	if len(src) < 8 {
		return ErrBadHeader
	}

	if string(src[0:8]) != "OpusTags" {
		return ErrBadHeader
	}

	return nil
}

type Reader struct {
	oggr           oggReader
	channels       int
	preSkipSamples int64
	gain           float32
	// tags           tags
	decoder     *opus.MSDecoder
	pcm         []byte // decoded page contents
	off         int    // offset within pcm for the next read
	nextPagePos int64  // granule position of the next page, in bytes
	err         error
}

func NewReader(r io.Reader) (*Reader, error) {
	var oggr oggReader
	oggr.source = r
	oggr.seeker, _ = r.(io.Seeker)

	p, err := oggr.NextPacket()
	if err != nil {
		return nil, err
	}
	var header header
	if err := parseHeader(p, &header); err != nil {
		return nil, err
	}

	p, err = oggr.NextPacket()
	if err != nil {
		return nil, err
	}
	var tags tags
	if err := parseTags(p, &tags); err != nil {
		return nil, err
	}

	decoder, err := opus.NewMSDecoder(48000, header.StreamCount, header.CoupledCount, header.Mapping[:header.ChannelCount])
	if err != nil {
		return nil, err
	}

	rr := &Reader{
		oggr:           oggr,
		channels:       header.ChannelCount,
		preSkipSamples: int64(header.PreSkip),
		gain:           float32(math.Pow(10, float64(header.OutputGain)/5120)),
		// tags:           tags,
		decoder: decoder,
	}

	// Seek to the actual start
	if _, err := rr.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return rr, nil
}

func (r *Reader) Channels() int {
	return r.channels
}

func (r *Reader) sampleSize() int {
	return r.channels * 4
}

func (r *Reader) preSkip() int64 {
	return r.preSkipSamples * int64(r.sampleSize())
}

func (r *Reader) SampleRate() int {
	return 48000
}

func (r *Reader) Read(b []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}

	if r.off >= len(r.pcm) {
		packet, err := r.oggr.NextPacket()
		if err != nil {
			r.err = err
			return 0, r.err
		}

		durSamples, err := packetDurationSamples(packet)
		if err != nil {
			r.err = err
			return 0, r.err
		}
		dur := durSamples * r.sampleSize()

		// If b is appropriately aligned and big enough to fit the decoded data,
		// decode directly into it.
		if uintptr(unsafe.Pointer(unsafe.SliceData(b)))%4 == 0 && len(b) >= dur {
			decodedSamples, err := multistreamDecodeAndApplyGain(r.decoder, packet, asFloatSlice(b), false, r.gain)
			if err != nil {
				r.err = err
				return 0, r.err
			}
			decoded := decodedSamples * r.sampleSize()
			r.nextPagePos += int64(decoded)
			return decoded, nil
		}

		pcm := r.pcm[:cap(r.pcm)]
		if len(pcm) < dur {
			pcm = make([]byte, dur)
		}
		decodedSamples, err := multistreamDecodeAndApplyGain(r.decoder, packet, asFloatSlice(pcm), false, r.gain)
		if err != nil {
			r.err = err
			return 0, r.err
		}
		decoded := decodedSamples * r.sampleSize()
		r.pcm = pcm[:decoded]
		r.off = 0
		r.nextPagePos += int64(decoded)
	}

	n := copy(b, r.pcm[r.off:])
	r.off += n
	return n, nil
}

func multistreamDecodeAndApplyGain(decoder *opus.MSDecoder, data []byte, pcm []float32, decodeFEC bool, gain float32) (int, error) {
	decoded, err := decoder.Decode(data, pcm, decodeFEC)
	if err == nil {
		for i := range pcm[:decoded] {
			pcm[i] *= gain
		}
	}
	return decoded, err
}

// Seek sets the offset, in bytes, for the next Read.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.pos() + offset
	case io.SeekEnd:
		// TODO: remember LastPosition
		endPosSamples, err := r.oggr.LastPosition()
		if err != nil {
			// BUG: we might not be able to continue after the error
			return r.pos(), err
		}
		abs = endPosSamples*int64(r.sampleSize()) + offset - r.preSkip()
	default:
		return r.pos(), errors.New("invalid whence")
	}
	if abs < 0 {
		return r.pos(), errors.New("seek to a negative position")
	}
	abs += r.preSkip()

	if r.pagePos() <= abs && abs <= r.nextPagePos {
		// Seeking within the current page.
		r.off = int(abs - r.pagePos())
		return r.pos(), nil
	}

	pagePosSamples, err := r.oggr.SeekPageBefore(abs / int64(r.sampleSize()))
	if err != nil {
		// BUG: we might not be able to continue after the error
		return r.pos(), err
	}
	pagePos := pagePosSamples * int64(r.sampleSize())

	r.pcm = r.pcm[:0]
	r.off = 0
	r.nextPagePos = pagePos

	// Skip until we get to abs
	_, err = io.CopyN(io.Discard, r, abs-pagePos)
	return r.pos(), err
}

// Position of the next read within the sample stream, in bytes, without the
// preSkip bytes.
func (r *Reader) pos() int64 {
	return r.pagePos() + int64(r.off) - r.preSkip()
}

func (r *Reader) pagePos() int64 {
	return r.nextPagePos - int64(len(r.pcm))
}

// packetDurationSamples returns number of per channel samples in an Opus packet.
func packetDurationSamples(packet []byte) (int, error) {
	frames, err := opus.PacketFrames(packet)
	if err != nil {
		return 0, err
	}
	frameSize, err := opus.PacketSamplesPerFrame(packet, 48000)
	if err != nil {
		return 0, err
	}
	samples := frames * frameSize
	if samples > 120*48 {
		return 0, ErrBadPacket
	}
	return frames * frameSize, nil
}

func asFloatSlice(s []byte) []float32 {
	return unsafe.Slice((*float32)(unsafe.Pointer(unsafe.SliceData(s))), len(s)/4)
}
