package nice

import (
	"io"
	"reflect"
)

type Decoder struct {
	scratch []byte

	r      io.Reader
	budget Budget

	getArshaler func(reflect.Type) arshaler
}

func NewDecoder(r io.Reader, opts ...Options) *Decoder {
	d := new(Decoder)
	d.Reset(r, opts...)
	return d
}

// Reset keeps the scratch buffer.
func (d *Decoder) Reset(r io.Reader, opts ...Options) {
	d.r = r

	collectedOptions := collectOptions(opts...)
	d.budget.reset(collectedOptions.budget)
	d.getArshaler = collectedOptions.getArshaler
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

func decodeBytes(dec *Decoder, b []byte) error {
	_, err := io.ReadFull(dec.Reader(), b)
	return err
}
