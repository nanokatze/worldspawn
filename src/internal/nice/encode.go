package nice

import (
	"io"
)

// TODO: track our position for error reporting.

type Encoder struct {
	Scratch []byte

	w    io.Writer
	opts options
}

func NewEncoder(w io.Writer, opts ...Option) *Encoder {
	// TODO: could we factor this out into a Reset method? It's not very clear
	// how to go about resetting opts in an "unsurprising" manner.
	e := new(Encoder)
	e.w = w
	for _, opt := range opts {
		opt(&e.opts)
	}
	return e
}

func (e *Encoder) Writer() io.Writer {
	return e.w
}
