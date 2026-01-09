// Package opusfile provides functions for reading and decoding Opus encapsulated in
// Ogg. See RFC7845 for details.
package opusfile

// TODO: clean up ogg code

import (
	"errors"
	"io"
	"math"
	"unsafe"

	"worldspawn/internal/fuckwwise/opus"
)

var ErrBadHeader = errors.New("invalid Ogg Opus header")

var ErrBadPacket = errors.New("packet failed to decode properly")

type Reader struct {
	pr             oggReader
	channels       int32
	preSkipSamples uint32
	gain           float32
	tags           *opusTags
	decoder        *opus.MSDecoder
	pcm            []byte // decoded page contents, will be cast to []float32 so must be appropriately aligned
	off            int    // offset within pcm for the next read
	nextPagePos    int64  // granule position of the next page, in bytes
	err            error
}

func NewReader(r io.Reader) (*Reader, error) {
	var oggr oggReader
	oggr.source = r
	oggr.seeker, _ = r.(io.Seeker)

	p, err := oggr.NextPacket()
	if err != nil {
		return nil, err
	}
	head, err := parseHead(p)
	if err != nil {
		return nil, err
	}

	p, err = oggr.NextPacket()
	if err != nil {
		return nil, err
	}
	tags, err := parseTags(p)
	if err != nil {
		return nil, err
	}

	decoder, err := opus.NewMSDecoder(48000, int(head.StreamCount), int(head.CoupledCount), head.Mapping[:head.ChannelCount])
	if err != nil {
		return nil, err
	}

	rr := &Reader{
		channels:       int32(head.ChannelCount),
		preSkipSamples: head.PreSkip,
		gain:           float32(math.Pow(10, float64(head.OutputGain)/5120)),
		tags:           tags,
		pr:             oggr,
		decoder:        decoder,
	}
	// Seek to the actual start
	if _, err := rr.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	return rr, nil
}

func (r *Reader) Channels() int {
	return int(r.channels)
}

func (r *Reader) SampleRate() int {
	return 48000
}

func (r *Reader) preSkip() int64 {
	return int64(r.preSkipSamples) * int64(r.sampleSize())
}

func (r *Reader) sampleSize() int {
	return r.Channels() * 4
}

func (r *Reader) Read(b []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}

	if r.off >= len(r.pcm) {
		packet, err := r.pr.NextPacket()
		if err != nil {
			return 0, err
		}

		durSamples, err := packetDurationSamples(packet)
		if err != nil {
			return 0, err
		}
		dur := durSamples * r.sampleSize()

		// Fast path: if b is appropriately aligned and big enough to fit the
		// decoded data, decode directly into it.
		if uintptr(unsafe.Pointer(unsafe.SliceData(b)))%4 == 0 && len(b) >= dur {
			decodedSamples, err := multistreamDecodeAndApplyGain(r.decoder, packet, asFloatSlice(b), false, r.gain)
			decoded := decodedSamples * r.sampleSize()
			if err != nil {
				return decoded, err
			}
			r.nextPagePos += int64(decoded)
			return decoded, nil
		}

		pcm := r.pcm[:cap(r.pcm)]
		if len(pcm) < dur {
			pcm = make([]byte, dur)
		}
		decodedSamples, err := multistreamDecodeAndApplyGain(r.decoder, packet, asFloatSlice(pcm), false, r.gain)
		decoded := decodedSamples * r.sampleSize()
		if err != nil {
			return decoded, err
		}
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
	for i := range pcm[:decoded] {
		pcm[i] *= gain
	}
	return decoded, err
}

// Seek sets the offset, in bytes, for the next Read.
func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	if r.err != nil {
		return r.pos(), r.err
	}

	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.pos() + offset
	case io.SeekEnd:
		// TODO: poke LastPosition once
		endPosSamples, err := r.pr.LastPosition()
		if err != nil {
			// BUG: the error might be unrecoverable
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

	pagePosSamples, err := r.pr.SeekPageBefore(abs / int64(r.sampleSize()))
	if err != nil {
		// BUG: the error might be unrecoverable
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
	p := unsafe.Pointer(unsafe.SliceData(s))
	if uintptr(p)%4 != 0 {
		panic("misaligned pointer")
	}
	return unsafe.Slice((*float32)(p), len(s)/4)
}
