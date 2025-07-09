// Package wav implements Waveform Audio File Format ReaderAt. Only 16-bit LPCM
// is supported. TODO: expand on support for other formats?
package wav

// http://soundfile.sapp.org/doc/WaveFormat/

type _WAVEFORMAT struct {
	FormatTag      uint16
	Channels       uint16
	SamplesPerSec  uint32
	AvgBytesPerSec uint32
	BlockAlign     uint16
	BitsPerSample  uint16
}

type chunk struct {
	Id     [4]byte
	Size   uint32
	Format [4]byte
}

type subchunk struct {
	Id   [4]byte
	Size uint32
}
