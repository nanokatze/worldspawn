// Package wav implements Waveform Audio File Format ReaderAt. Only 16-bit LPCM
// is supported. TODO: expand on support for other formats?
package wav

// http://soundfile.sapp.org/doc/WaveFormat/

const (
	_WAVE_FORMAT_PCM        = 1
	_WAVE_FORMAT_IEEE_FLOAT = 3
)

type _WAVEFORMAT struct {
	FormatTag      uint16
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
}

type _PCMWAVEFORMAT struct {
	_WAVEFORMAT
	BitsPerSample uint16
}
