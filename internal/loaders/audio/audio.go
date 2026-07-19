package audio

import (
	"errors"
	"io"
	"strings"
)

type format struct {
	magic     string
	newReader func(r io.ReaderAt) (Reader, error)
}

// TODO: speed this up somehow and make it thread safe.
var formats []format

func RegisterFormat(name, magic string, newReader func(r io.ReaderAt) (Reader, error)) {
	formats = append(formats, format{magic, newReader})
}

type Config struct {
	Format     int
	Channels   int
	SampleRate int
}

type Reader interface {
	Config() Config
	io.Reader
}

// TODO: should accept plain io.Reader.
func NewReader(r io.ReaderAt) (Reader, error) {
	var magic [100]byte
	if _, err := r.ReadAt(magic[:], 0); err != nil {
		return nil, err
	}

	// TODO: actually autodetect format
	for _, format := range formats {
		if strings.HasPrefix(string(magic[:]), format.magic) {
			return format.newReader(r)
		}
	}

	return nil, errors.New("unrecognized format")
}
