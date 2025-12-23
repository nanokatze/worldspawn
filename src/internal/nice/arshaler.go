package nice

import "reflect"

type (
	marshaler   func(enc *Encoder, v reflect.Value) error
	unmarshaler func(dec *Decoder, v reflect.Value) error
)

type arshaler struct {
	marshal   marshaler
	unmarshal unmarshaler
}
