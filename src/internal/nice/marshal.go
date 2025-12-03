package nice

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"sync"
)

func MarshalEncode(enc *Encoder, in any) error {
	p := reflect.ValueOf(in)
	if p.Kind() != reflect.Pointer || p.IsNil() {
		panic("expecting a non-nil pointer")
	}

	v := p.Elem()

	t := v.Type()

	marshal := mapGetOrElse(enc.customMarshalers, t, func() marshaler { return defaultMarshaler(t) })
	return marshal(enc, v)
}

var defaultMarshalers sync.Map

func defaultMarshaler(t reflect.Type) marshaler {
	if m, ok := defaultMarshalers.Load(t); ok {
		return m.(marshaler)
	}
	return defaultMarshalerSlow(t)
}

func defaultMarshalerSlow(t reflect.Type) marshaler {
	m, _ := defaultMarshalers.LoadOrStore(t, makeDefaultMarshaler(t))
	return m.(marshaler)
}

func makeDefaultMarshaler(t reflect.Type) marshaler {
	// NOTE: when implementing BinaryAppender and BinaryMarshaler we'll need to
	// encode length, as UnmarshalBinary doesn't return the length it consumed.
	// NOTE: we'll also need to distinguish whether the default marshaler is
	// really "default" in the sense that we can use optimized marshaler.

	switch t.Kind() {
	case reflect.Bool:
		return func(enc *Encoder, v reflect.Value) error {
			return writeBool(enc, v.Bool())
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := int(t.Size())
		if t.Kind() == reflect.Int {
			n = 8
		}
		return func(enc *Encoder, v reflect.Value) error {
			return writeUint(enc, uint64(v.Int()), n)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n := int(t.Size())
		if t.Kind() == reflect.Uint {
			n = 8
		}
		return func(enc *Encoder, v reflect.Value) error {
			return writeUint(enc, v.Uint(), n)
		}

	case reflect.Float32:
		return func(enc *Encoder, v reflect.Value) error {
			x := uint64(math.Float32bits(float32(v.Float())))
			return writeUint(enc, x, 4)
		}

	case reflect.Float64:
		return func(enc *Encoder, v reflect.Value) error {
			x := math.Float64bits(v.Float())
			return writeUint(enc, x, 8)
		}

	case reflect.Array:
		n := t.Len()
		if n == 0 {
			return marshalEmpty
		}

		defaultMarshal := defaultMarshaler(t.Elem())

		return func(enc *Encoder, v reflect.Value) error {
			// TODO: rename ok to something else, e.g. ok, haveCustomMarshal
			marshal, ok := enc.customMarshalers[t.Elem()]
			if !ok {
				marshal = defaultMarshal
			}

			if !ok {
				if optimizedArshalers {
					// TODO: specialized array arshalers
					// NOTE: https://github.com/golang/go/issues/27727
				}
			}
			for i := range n {
				if err := marshal(enc, v.Index(i)); err != nil {
					return err
				}
			}
			return nil
		}

	case reflect.Map:
		defaultMarshalKey := defaultMarshaler(t.Key())
		defaultMarshalVal := defaultMarshaler(t.Elem())

		return func(enc *Encoder, m reflect.Value) error {
			n := m.Len()
			if err := writeLen(enc, n); err != nil {
				return err
			}
			if n == 0 {
				return nil
			}

			marshalKey := mapGetOrDefault(enc.customMarshalers, t.Key(), defaultMarshalKey)
			marshalVal := mapGetOrDefault(enc.customMarshalers, t.Elem(), defaultMarshalVal)

			k := reflect.New(t.Key()).Elem()
			v := reflect.New(t.Elem()).Elem()
			for iter := m.MapRange(); iter.Next(); {
				if n == 0 {
					panic("unreachable") // TODO: a more informative message
				}
				n--

				k.SetIterKey(iter)
				v.SetIterValue(iter)
				if err := marshalKey(enc, k); err != nil {
					return err
				}
				if err := marshalVal(enc, v); err != nil {
					return err
				}
			}
			return nil
		}

	case reflect.Pointer:
		defaultMarshal := defaultMarshaler(t.Elem())

		// TODO: we could have a field option to assume the field is never nil
		// so it's marshaled trivially

		return func(enc *Encoder, v reflect.Value) error {
			isNonNil := !v.IsNil()
			if err := writeBool(enc, isNonNil); err != nil {
				return err
			}
			if !isNonNil {
				return nil
			}

			marshal := mapGetOrDefault(enc.customMarshalers, t.Elem(), defaultMarshal)
			return marshal(enc, v.Elem())
		}

	case reflect.Slice:
		defaultMarshal := defaultMarshaler(t.Elem())

		// TODO: unify with array marshaler to the extent possible

		return func(enc *Encoder, v reflect.Value) error {
			n := v.Len()
			if err := writeLen(enc, n); err != nil {
				return err
			}
			if n == 0 {
				return nil
			}

			// TODO: enforce size limits when encoding too

			marshal := mapGetOrDefault(enc.customMarshalers, t.Elem(), defaultMarshal)
			for i := range n {
				if err := marshal(enc, v.Index(i)); err != nil {
					return err
				}
			}
			return nil
		}

	case reflect.String:
		return func(enc *Encoder, v reflect.Value) error {
			// TODO: optimize for non-StringWriters as well?
			// TODO: see if there's anything to do to optimize small string
			// writes
			s := v.String()
			// TODO: enforce size limit
			if err := writeLen(enc, len(s)); err != nil {
				return err
			}
			if _, err := io.WriteString(enc.Writer(), s); err != nil {
				return err
			}
			return nil
		}

	case reflect.Struct:
		fs := structInfo(t)
		if len(fs) == 0 {
			return marshalEmpty
		}

		return func(enc *Encoder, v reflect.Value) error {
			for _, f := range fs {
				marshal := mapGetOrDefault(enc.customMarshalers, f.Type, f.DefaultMarshal)
				if err := marshal(enc, v.Field(f.Index)); err != nil {
					return err
				}
			}
			return nil
		}

	default:
		err := fmt.Errorf("no default marshaler for %v", t)
		return func(enc *Encoder, v reflect.Value) error {
			return err
		}
	}
}

func marshalEmpty(enc *Encoder, v reflect.Value) error {
	return nil
}

func writeBytes(enc *Encoder, b []byte) error {
	_, err := enc.Writer().Write(b)
	return err
}

func writeUint(enc *Encoder, x uint64, n int) error {
	enc.Scratch = binary.LittleEndian.AppendUint64(enc.Scratch[:0], x)
	return writeBytes(enc, enc.Scratch[:n])
}

func writeBool(enc *Encoder, x bool) error {
	b := byte(0)
	if x {
		b = 1
	}
	enc.Scratch = append(enc.Scratch[:0], b)
	return writeBytes(enc, enc.Scratch)
}

func writeLen(enc *Encoder, x int) error {
	if x < 0 {
		panic("bad")
	}
	return writeUint(enc, uint64(x), 8)
}
