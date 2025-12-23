package nice

import (
	"io"
	"reflect"
)

type Encoder struct {
	Scratch []byte

	w io.Writer

	// TODO: can we thread this in a way that's decoupled from Encoder?
	getArshaler func(reflect.Type) arshaler
}

func NewEncoder(w io.Writer, opts ...Options) *Encoder {
	e := new(Encoder)
	e.Reset(w, opts...)
	return e
}

// Reset keeps the scratch buffer.
func (e *Encoder) Reset(w io.Writer, opts ...Options) {
	e.w = w

	collectedOptions := collectOptions(opts...)
	e.getArshaler = collectedOptions.getArshaler
}

func (e *Encoder) Writer() io.Writer {
	return e.w
}

func encodeBytes(enc *Encoder, b []byte) error {
	_, err := enc.Writer().Write(b)
	return err
}
