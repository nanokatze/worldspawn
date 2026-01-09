package nice

import (
	"bytes"
	"reflect"
)

func Unmarshal(in []byte, out any, opts ...Options) error {
	buf := bytes.NewReader(in)

	if err := UnmarshalDecode(NewDecoder(buf, opts...), out); err != nil {
		return err
	}
	return nil
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
