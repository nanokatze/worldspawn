package proto

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"sync"

	"worldspawn/internal/proto/protowire"
)

var defaultArshalers sync.Map

func getDefaultArshaler(t reflect.Type) arshaler {
	if a, ok := defaultArshalers.Load(t); ok {
		return a.(arshaler)
	}
	return getDefaultArshalerSlow(t)
}

func getDefaultArshalerSlow(t reflect.Type) arshaler {
	a, _ := defaultArshalers.LoadOrStore(t, makeDefaultArshaler(t))
	return a.(arshaler)
}

func makeDefaultArshaler(t reflect.Type) arshaler {
	// TODO: handle encoding.BinaryMarshaler and encoding.BinaryUnmarshaler here

	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return varintArshaler

	case reflect.String:
		return arshaler{
			marshal: func(enc *Encoder, scratch []byte, v reflect.Value) (protowire.Value, error) {
				scratch = append(scratch, v.String()...)
				return protowire.Value{protowire.TypeBytes, scratch}, nil
			},
			unmarshal: func(dec *Decoder, w protowire.Value, v reflect.Value) error {
				v.SetString(string(w.Payload))
				return nil
			},
		}

	case reflect.Struct:
		return makeDefaultStructArshaler(t)

	default:
		err := fmt.Errorf("no default arshaler for type %v", t) // TODO: make a type for this error?
		return arshaler{
			marshal: func(_ *Encoder, _ []byte, _ reflect.Value) (protowire.Value, error) {
				return protowire.Value{}, err
			},
			unmarshal: func(_ *Decoder, _ protowire.Value, _ reflect.Value) error {
				return err
			},
		}
	}
}

// TODO: move to its own file?
var varintArshaler = arshaler{
	marshal: func(enc *Encoder, scratch []byte, v reflect.Value) (protowire.Value, error) {
		var _int uint64
		switch {
		case v.CanInt():
			_int = uint64(v.Int())
		case v.CanUint():
			_int = v.Uint()
		default:
			panic("unreachable")
		}
		return protowire.Value{protowire.TypeVarint, protowire.AppendVarint(scratch, _int)}, nil
	},
	unmarshal: func(dec *Decoder, w protowire.Value, v reflect.Value) error {
		if w.Type != protowire.TypeVarint {
			return errors.New("mismatched wire type")
		}
		_int, n := protowire.ConsumeVarint(w.Payload)
		if n <= 0 {
			return errors.New("parsing int failed")
		}
		// TODO: check that it's in range of the target type? Ok apparently
		// official impl doesn't do that either...
		switch {
		case v.CanInt():
			v.SetInt(int64(_int))
		case v.CanUint():
			v.SetUint(_int)
		default:
			// TODO: apparently the official impl simply doesn't complain if the
			// types mismatch
			return errors.New("can't unmarshal varint into whatever")
		}
		return nil
	},
}

func makeDefaultStructArshaler(s reflect.Type) arshaler {
	// TODO: field numbers should be specified with an explicit tag
	// TODO: do more work at this stage
	fields := slices.Clip(slices.Collect(s.Fields()))

	return arshaler{
		marshal: func(enc *Encoder, scratch []byte, v reflect.Value) (protowire.Value, error) {
			b := protowire.MessageBuilder(scratch)
			for _, field := range fields {
				wireFieldNumber := field.Index[0] + 1

				// TODO: implement omitempty semantics?

				// TODO: guess scratch size from wireFieldNumber and field.Type.

				marshal := enc.opts.getArshaler(field.Type).marshal
				marshaled, err := marshal(enc, b.ScratchBuffer(protowire.MaxRecordHeaderLen), v.Field(field.Index[0]))
				if err != nil {
					return protowire.Value{}, err
				}
				b.AppendRecord(protowire.Record{protowire.PackTag(wireFieldNumber, marshaled.Type), marshaled.Payload})
			}
			return protowire.Value{protowire.TypeBytes, b}, nil
		},
		unmarshal: func(dec *Decoder, w protowire.Value, v reflect.Value) error {
			// TODO: validate that w.Type == wireTypeBytes
			if w.Type != protowire.TypeBytes {
				return errors.New("expecting bytes wire type")
			}

			p := protowire.MessageParser(w.Payload)
			for len(p) > 0 {
				record, err := p.ConsumeRecord()
				if err != nil {
					return err
				}

				// TODO: if we don't have this field, skip it. Actually we might
				// need to do work to implement default values or whatever.

				hostField := v.Field(record.FieldNumber() - 1)

				arshaler := dec.opts.getArshaler(hostField.Type())

				if err := arshaler.unmarshal(dec, record.Value(), hostField); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
