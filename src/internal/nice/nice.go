package nice

import (
	"reflect"
)

// TODO: respect types implementing encoding.BinaryAppender and
// encoding.BinaryMarshaler and their respective unmarshaling parts

// TODO: encoding/binary is currently faster than us in some cases. We should
// analyze why and see where we can improve.

// TODO: explore optimizing un/marshaling of certain structs with a memcpy

// TODO: schema?

var optimizedArshalers = false

type (
	marshaler   func(enc *Encoder, v reflect.Value) error
	unmarshaler func(dec *Decoder, v reflect.Value) error
)

// TODO: rename
type fieldInfo struct {
	Type  reflect.Type
	Index int
	// These should be put into their own arrays tbf...
	DefaultMarshal   marshaler
	DefaultUnmarshal unmarshaler
}

// TODO: rename
func structInfo(t reflect.Type) []fieldInfo {
	var fs []fieldInfo
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		fs = append(fs, fieldInfo{
			Type:             f.Type,
			Index:            f.Index[0], // TODO: handle embedded structs etc
			DefaultMarshal:   defaultMarshaler(f.Type),
			DefaultUnmarshal: defaultUnmarshaler(f.Type),
		})
	}
	return fs
}
