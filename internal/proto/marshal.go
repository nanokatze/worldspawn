package proto

import (
	"errors"
	"reflect"

	"worldspawn/internal/proto/protowire"
)

func Unmarshal(in []byte, out any) error {
	opts := &options{}

	v := reflect.ValueOf(out).Elem()

	unmarshal := opts.getArshaler(v.Type()).unmarshal
	if err := unmarshal(&Decoder{opts: opts}, protowire.Value{Type: protowire.TypeBytes, Payload: in}, v); err != nil {
		return err
	}

	return nil
}

// TODO: make this have append semantics
func Append(out []byte, in any) ([]byte, error) {
	opts := &options{}

	v := reflect.ValueOf(in).Elem()

	scratch := out[len(out):]

	marshal := opts.getArshaler(v.Type()).marshal
	marshaled, err := marshal(&Encoder{opts: opts}, scratch, v)
	if err != nil {
		return nil, err
	}
	if marshaled.Type != protowire.TypeBytes {
		return nil, errors.New("sus")
	}

	return append(out, marshaled.Payload...), nil
}
