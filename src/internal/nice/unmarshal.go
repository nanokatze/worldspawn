package nice

import (
	"bytes"
	"reflect"
)

// TODO: rename Unmarshal to just Decode? Also add a convenience DecodeBytes or
// w/e idk
func UnmarshalDecode(dec *Decoder, out any) error {
	p := reflect.ValueOf(out)
	if p.Kind() != reflect.Pointer || p.IsNil() {
		panic("expecting a non-nil pointer")
	}

	v := p.Elem()

	t := v.Type()

	unmarshal := dec.getArshaler(t).unmarshal
	return unmarshal(dec, v)
}

func Unmarshal(in []byte, out any, opts ...Option) error {
	buf := bytes.NewReader(in)

	if err := UnmarshalDecode(NewDecoder(buf, opts...), out); err != nil {
		return err
	}
	return nil
}
