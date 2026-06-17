package nice

import (
	"bytes"
	"io"
	"reflect"
)

// TODO: pool the decoders and the byte buffers

func Unmarshal(in []byte, out any, opts ...Options) error {
	dec := new(Decoder)
	dec.Reset(bytes.NewBuffer(in), opts...)
	if err := UnmarshalDecode(dec, out); err != nil {
		return err
	}
	return nil
}

func UnmarshalRead(r io.Reader, out any, opts ...Options) error {
	dec := new(Decoder)
	dec.Reset(r, opts...)
	return UnmarshalDecode(dec, out)
}

func UnmarshalDecode(dec *Decoder, out any) error {
	p := reflect.ValueOf(out)
	if p.Kind() != reflect.Pointer || p.IsNil() {
		panic("expecting a non-nil pointer")
	}

	v := p.Elem()
	v.SetZero()

	t := v.Type()

	unmarshal := dec.getArshaler(t).unmarshal
	return unmarshal(dec, v)
}
