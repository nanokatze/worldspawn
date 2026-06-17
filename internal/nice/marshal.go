package nice

import (
	"bytes"
	"io"
	"reflect"
)

// TODO: pool the encoders and the byte buffers

func Marshal(in any, opts ...Options) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := new(Encoder)
	enc.Reset(buf, opts...)
	if err := MarshalEncode(enc, in); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func MarshalWrite(w io.Writer, in any, opts ...Options) error {
	enc := new(Encoder)
	enc.Reset(w, opts...)
	return MarshalEncode(enc, in)
}

// TODO: should this take opts too?
func MarshalEncode(enc *Encoder, in any) error {
	p := reflect.ValueOf(in)
	if p.Kind() != reflect.Pointer || p.IsNil() {
		panic("expecting a non-nil pointer")
	}

	v := p.Elem()

	t := v.Type()

	marshal := enc.getArshaler(t).marshal
	return marshal(enc, v)
}
