package nice

import (
	"io"
	"reflect"
)

type Decoder struct {
	scratch []byte

	r      io.Reader
	budget Budget

	customUnmarshalers map[reflect.Type]unmarshaler
}

func NewDecoder(r io.Reader, opts ...Option) *Decoder {
	d := new(Decoder)
	d.Reset(r, opts...)
	return d
}

// Reset keeps the scratch buffer.
func (d *Decoder) Reset(r io.Reader, opts ...Option) {
	collectedOpts := collectOptions(opts...)

	d.r = r
	d.budget.reset(collectedOpts.memoryLimit)
	d.customUnmarshalers = collectedOpts.customUnmarshalers
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

func (d *Decoder) Budget() *Budget {
	return &d.budget
}
