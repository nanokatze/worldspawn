package nice

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

type Decoder struct {
	scratch []byte

	r      io.Reader
	budget Budget

	getArshaler func(reflect.Type) arshaler
}

func NewDecoder(r io.Reader, opts ...Option) *Decoder {
	d := new(Decoder)
	d.Reset(r, opts...)
	return d
}

// Reset keeps the scratch buffer.
func (d *Decoder) Reset(r io.Reader, opts ...Option) {
	collectedOptions := collectOptions(opts...)

	d.r = r
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

func readBytes(dec *Decoder, b []byte) error {
	_, err := io.ReadFull(dec.Reader(), b)
	return err
}

func readUint(dec *Decoder, n int) (uint64, error) {
	buf := dec.Scratch(8)
	clear(buf)
	if err := readBytes(dec, buf[:n]); err != nil {
		return 0, nil
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func readBool(dec *Decoder) (bool, error) {
	x, err := readUint(dec, 1)
	if err != nil {
		return false, err
	}
	if x != 0 && x != 1 {
		return false, fmt.Errorf("bad") // TODO: nicer error message
	}
	return x != 0, nil
}

func readLen(dec *Decoder) (int, error) {
	x, err := readUint(dec, 8)
	if err != nil {
		return -1, err
	}
	if x != uint64(int(x)) {
		return -1, fmt.Errorf("bad")
	}
	if int(x) < 0 {
		return -1, fmt.Errorf("bad")
	}
	return int(x), nil
}
