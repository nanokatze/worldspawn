package nice

import (
	"bytes"
	"reflect"
)

func Marshal(in any, opts ...Options) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	if err := MarshalEncode(NewEncoder(buf, opts...), in); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

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
