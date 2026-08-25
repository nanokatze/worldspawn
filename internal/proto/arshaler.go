package proto

import (
	"reflect"

	"worldspawn/internal/proto/protowire"
)

type (
	marshaler   func(enc *Encoder, scratch []byte, v reflect.Value) (protowire.Value, error)
	unmarshaler func(dec *Decoder, w protowire.Value, v reflect.Value) error
)

type arshaler struct {
	marshal   marshaler
	unmarshal unmarshaler
}
