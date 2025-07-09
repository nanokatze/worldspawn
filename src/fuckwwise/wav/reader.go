package wav

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

// TODO: change this to take a Reader and return a ReaderAt opportunistically?

/*
type Reader interface {
	io.Reader
	io.Seeker
	io.Closer
	Format() int
	Channels() int
	SampleRate() int
}
*/

type Reader struct {
	r          *io.SectionReader
	channels   int
	sampleRate int
}

func NewReader(r io.ReaderAt) (*Reader, error) {
	sr := io.NewSectionReader(r, 0, math.MaxInt64)

	var header chunk
	if err := binary.Read(sr, binary.LittleEndian, &header); err != nil {
		return nil, err
	}
	if string(header.Id[:]) != "RIFF" {
		return nil, errors.New("not a WAVE file")
	}
	if string(header.Format[:]) != "WAVE" {
		return nil, errors.New("not a WAVE file")
	}

	var waveformat *_WAVEFORMAT

	for {
		var subchunkHeader subchunk
		if err := binary.Read(sr, binary.LittleEndian, &subchunkHeader); err != nil {
			return nil, err
		}

		off, _ := sr.Seek(0, io.SeekCurrent)

		switch string(subchunkHeader.Id[:]) {
		default:
			// unrecognized subchunk

		case "fmt ":
			if waveformat != nil {
				return nil, errors.New("duplicate fmt subchunk")
			}

			waveformat = new(_WAVEFORMAT)
			if err := binary.Read(sr, binary.LittleEndian, waveformat); err != nil {
				return nil, err
			}

			// ffmpeg sets fmt_.FormatTag to 0xfffe whenever we want
			// an unusual sampling rate; the actual format is then
			// set in subFormat field of WAVEFORMATEXTENSIBLE. We
			// don't handle that for now.

		case "data":
			if waveformat == nil {
				return nil, errors.New("expecting fmt subchunk before data")
			}

			if waveformat.SamplesPerSec < 1 {
				return nil, fmt.Errorf("invalid sample rate %d", waveformat.SamplesPerSec)
			}
			if waveformat.Channels < 1 {
				return nil, fmt.Errorf("invalid channel count %d", waveformat.Channels)
			}
			// We only support 16-bit samples.
			if waveformat.BitsPerSample != 16 {
				return nil, fmt.Errorf("unsupported bits per sample %d", waveformat.BitsPerSample)
			}

			return &Reader{
				r:          io.NewSectionReader(r, off, int64(subchunkHeader.Size)),
				channels:   int(waveformat.Channels),
				sampleRate: int(waveformat.SamplesPerSec),
			}, nil
		}

		sr.Seek(off+int64(subchunkHeader.Size), io.SeekStart)
	}
}

func (r *Reader) Format() int {
	return 1
}

func (r *Reader) Channels() int {
	return r.channels
}

func (r *Reader) SampleRate() int {
	return r.sampleRate
}

func (r *Reader) Read(b []byte) (n int, err error) {
	return r.r.Read(b)
}

func (r *Reader) ReadAt(b []byte, off int64) (n int, err error) {
	return r.r.ReadAt(b, off)
}
