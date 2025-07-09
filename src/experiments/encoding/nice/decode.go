package nice

import "io"

type Decoder struct {
	scratch []byte

	r    io.Reader
	opts options
}

func NewDecoder(r io.Reader, opts ...Option) *Decoder {
	// TODO: could we factor this out into a Reset method? It's not very clear
	// how to go about resetting opts in an "unsurprising" manner.
	d := new(Decoder)
	d.r = r
	for _, opt := range opts {
		opt(&d.opts)
	}
	return d
}

func (d *Decoder) Scratch(n int) []byte {
	if n > cap(d.scratch) {
		// Use the append-make pattern so that runtime chooses a nice capacity
		// for us.
		d.scratch = append([]byte(nil), make([]byte, n)...)
	}
	return d.scratch[:n]
}

func (d *Decoder) Reader() io.Reader {
	return d.r
}
