package nice

import (
	"io"
	"reflect"
)

// TODO: track our position for error reporting.

type Encoder struct {
	Scratch []byte

	w io.Writer

	customMarshalers map[reflect.Type]marshaler
}

func NewEncoder(w io.Writer, opts ...Option) *Encoder {
	e := new(Encoder)
	e.Reset(w, opts...)
	return e
}

// Reset keeps the scratch buffer.
func (e *Encoder) Reset(w io.Writer, opts ...Option) {
	collectedOpts := collectOptions(opts...)

	e.w = w
	e.customMarshalers = collectedOpts.customMarshalers
}

func (e *Encoder) Writer() io.Writer {
	return e.w
}
