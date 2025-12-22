package nice

import (
	"encoding/binary"
	"io"
	"reflect"
)

// TODO: track our position for error reporting.

type Encoder struct {
	Scratch []byte

	w io.Writer

	// TODO: can we thread this in a way that's decoupled from Encoder?
	getArshaler func(reflect.Type) arshaler
}

func NewEncoder(w io.Writer, opts ...Option) *Encoder {
	e := new(Encoder)
	e.Reset(w, opts...)
	return e
}

// Reset keeps the scratch buffer.
func (e *Encoder) Reset(w io.Writer, opts ...Option) {
	collectedOptions := collectOptions(opts...)

	e.w = w
	e.getArshaler = collectedOptions.getArshaler
}

func (e *Encoder) Writer() io.Writer {
	return e.w
}

func writeBytes(enc *Encoder, b []byte) error {
	_, err := enc.Writer().Write(b)
	return err
}

// TODO: change n to be in number of bits?
func writeUint(enc *Encoder, x uint64, n int) error {
	enc.Scratch = binary.LittleEndian.AppendUint64(enc.Scratch[:0], x)
	return writeBytes(enc, enc.Scratch[:n])
}

func writeBool(enc *Encoder, x bool) error {
	b := byte(0)
	if x {
		b = 1
	}
	enc.Scratch = append(enc.Scratch[:0], b)
	return writeBytes(enc, enc.Scratch)
}

// TODO: let the user specify whether ints (and thus lengths) should be
// marshaled as 32 or 64 bits? Or we could encode ints as varints but that's
// ass.
func writeLen(enc *Encoder, x int) error {
	if x < 0 {
		panic("bad")
	}
	return writeUint(enc, uint64(x), 8)
}
