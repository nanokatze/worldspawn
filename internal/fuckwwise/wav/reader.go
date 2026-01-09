package wav

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"strconv"

	sfx "worldspawn/internal/fuckwwise"
	"worldspawn/internal/fuckwwise/wav/internal/riff"
)

type redundantChunkError [4]byte

func (e redundantChunkError) Error() string {
	return fmt.Sprintf("redundant %q chunk", string(e[:]))
}

type unsupportedBitsPerSampleError uint16

func (e unsupportedBitsPerSampleError) Error() string {
	return "unsupported bits per sample " + strconv.FormatInt(int64(e), 10)
}

type config struct {
	format     sfx.Format
	channels   int32
	sampleRate uint32
}

type Reader struct {
	r    *io.SectionReader
	conf config
}

func NewReader(r io.ReaderAt) (*Reader, error) {
	sr := io.NewSectionReader(r, 0, math.MaxInt64)

	var riffHeader riff.Chunk
	if err := binary.Read(sr, binary.LittleEndian, &riffHeader); err != nil {
		return nil, err
	}
	if string(riffHeader.Id[:]) != "RIFF" {
		return nil, errors.New("not a WAVE file")
	}
	if string(riffHeader.Format[:]) != "WAVE" {
		return nil, errors.New("not a WAVE file")
	}

	var conf config
	for {
		var header riff.Subchunk
		if err := binary.Read(sr, binary.LittleEndian, &header); err != nil {
			return nil, err
		}

		off, _ := sr.Seek(0, io.SeekCurrent)

		data := io.NewSectionReader(r, off, int64(header.Size))

		switch string(header.Id[:]) {
		case "fmt ":
			if conf != (config{}) {
				return nil, redundantChunkError(header.Id)
			}

			var wavefmt _WAVEFORMAT
			if err := binary.Read(data, binary.LittleEndian, &wavefmt); err != nil {
				return nil, err
			}
			data.Seek(0, io.SeekStart)

			if wavefmt.Channels < 1 {
				return nil, fmt.Errorf("unsupported channel count %d", wavefmt.Channels)
			}
			if wavefmt.SamplesPerSec < 1 {
				return nil, fmt.Errorf("unsupported sample rate %d", wavefmt.SamplesPerSec)
			}

			var format sfx.Format
			switch wavefmt.FormatTag {
			case _WAVE_FORMAT_PCM:
				var wavefmt _PCMWAVEFORMAT
				if err := binary.Read(data, binary.LittleEndian, &wavefmt); err != nil {
					return nil, err
				}

				switch wavefmt.BitsPerSample {
				case 16:
					format = sfx.FORMAT_S16
				default:
					return nil, unsupportedBitsPerSampleError(wavefmt.BitsPerSample)
				}

			case _WAVE_FORMAT_IEEE_FLOAT:
				var wavefmt _PCMWAVEFORMAT
				if err := binary.Read(data, binary.LittleEndian, &wavefmt); err != nil {
					return nil, err
				}

				switch wavefmt.BitsPerSample {
				case 32:
					format = sfx.FORMAT_F32
				default:
					return nil, unsupportedBitsPerSampleError(wavefmt.BitsPerSample)
				}

			default:
				return nil, fmt.Errorf("unsupported format tag 0x%x", wavefmt.FormatTag)
			}

			conf = config{
				format:     format,
				channels:   int32(wavefmt.Channels),
				sampleRate: wavefmt.SamplesPerSec,
			}

		case "data":
			if conf == (config{}) {
				return nil, errors.New("no fmt chunk")
			}

			return &Reader{
				r:    data,
				conf: conf,
			}, nil

		default:
		}

		sr.Seek(off+int64(header.Size), io.SeekStart)
	}
}

func (r *Reader) Format() sfx.Format {
	return r.conf.format
}

func (r *Reader) Channels() int {
	return int(r.conf.channels)
}

func (r *Reader) SampleRate() int {
	return int(r.conf.sampleRate)
}

func (r *Reader) Read(b []byte) (n int, err error) {
	return r.r.Read(b)
}

func (r *Reader) Seek(offset int64, whence int) (int64, error) {
	return r.r.Seek(offset, whence)
}

func (r *Reader) ReadAt(b []byte, off int64) (n int, err error) {
	return r.r.ReadAt(b, off)
}
