package nice

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"sync"
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

	unmarshal := mapGetOrElse(dec.customUnmarshalers, t, func() unmarshaler { return defaultUnmarshaler(t) })
	return unmarshal(dec, v)
}

func Unmarshal(in []byte, out any, opts ...Option) error {
	buf := bytes.NewReader(in)

	if err := UnmarshalDecode(NewDecoder(buf, opts...), out); err != nil {
		return err
	}
	return nil
}

var defaultUnmarshalers sync.Map

func defaultUnmarshaler(t reflect.Type) unmarshaler {
	if m, ok := defaultUnmarshalers.Load(t); ok {
		return m.(unmarshaler)
	}
	return defaultUnmarshalerSlow(t)
}

func defaultUnmarshalerSlow(t reflect.Type) unmarshaler {
	u, _ := defaultUnmarshalers.LoadOrStore(t, makeDefaultUnmarshaler(t))
	return u.(unmarshaler)
}

func makeDefaultUnmarshaler(t reflect.Type) unmarshaler {
	switch t.Kind() {
	case reflect.Bool:
		return func(dec *Decoder, v reflect.Value) error {
			b, err := readBool(dec)
			if err != nil {
				return err
			}
			v.SetBool(b)
			return nil
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := int(t.Size())
		if t.Kind() == reflect.Int {
			n = 8
		}
		return func(dec *Decoder, v reflect.Value) error {
			x, err := readUint(dec, n)
			if err != nil {
				return err
			}
			v.SetInt(int64(x))
			return nil
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n := int(t.Size())
		if t.Kind() == reflect.Uint {
			n = 8
		}
		return func(dec *Decoder, v reflect.Value) error {
			x, err := readUint(dec, n)
			if err != nil {
				return err
			}
			v.SetUint(x)
			return nil
		}

	case reflect.Float32:
		return func(dec *Decoder, v reflect.Value) error {
			x, err := readUint(dec, 4)
			if err != nil {
				return err
			}
			v.SetFloat(float64(math.Float32frombits(uint32(x))))
			return nil
		}

	case reflect.Float64:
		return func(dec *Decoder, v reflect.Value) error {
			x, err := readUint(dec, 8)
			if err != nil {
				return err
			}
			v.SetFloat(math.Float64frombits(x))
			return nil
		}

	case reflect.Array:
		n := t.Len()
		if n == 0 {
			return unmarshalEmpty
		}

		defaultUnmarshal := defaultUnmarshaler(t.Elem())

		return func(dec *Decoder, v reflect.Value) error {
			unmarshal, ok := dec.customUnmarshalers[t.Elem()]
			if !ok {
				unmarshal = defaultUnmarshal
			}

			if !ok {
				if optimizedArshalers {
					// TODO: specialized array arshalers
					// NOTE: https://github.com/golang/go/issues/27727
				}
			}
			for i := range n {
				if err := unmarshal(dec, v.Index(i)); err != nil {
					return err
				}
			}
			return nil
		}

	case reflect.Map:
		defaultUnmarshalKey := defaultUnmarshaler(t.Key())
		defaultUnmarshalVal := defaultUnmarshaler(t.Elem())

		return func(dec *Decoder, m reflect.Value) error {
			n, err := readLen(dec)
			if err != nil {
				return err
			}
			if n == 0 {
				m.SetZero()
				return nil
			}

			if err := accountForType(dec.Budget(), t.Key(), n); err != nil {
				return err
			}
			if err := accountForType(dec.Budget(), t.Elem(), n); err != nil {
				return err
			}

			m.Set(reflect.MakeMapWithSize(t, n))

			unmarshalKey := mapGetOrDefault(dec.customUnmarshalers, t.Key(), defaultUnmarshalKey)
			unmarshalVal := mapGetOrDefault(dec.customUnmarshalers, t.Elem(), defaultUnmarshalVal)

			k := reflect.New(t.Key()).Elem()
			v := reflect.New(t.Elem()).Elem()
			for range n {
				if err := unmarshalKey(dec, k); err != nil {
					return err
				}
				if err := unmarshalVal(dec, v); err != nil {
					return err
				}
				m.SetMapIndex(k, v)
			}
			return nil
		}

	case reflect.Pointer:
		defaultUnmarshal := defaultUnmarshaler(t.Elem())

		return func(dec *Decoder, v reflect.Value) error {
			isNonNil, err := readBool(dec)
			if err != nil {
				return err
			}
			if !isNonNil {
				v.SetZero()
				return nil
			}

			if err := accountForType(dec.Budget(), t.Elem(), 1); err != nil {
				return err
			}

			v.Set(reflect.New(t.Elem()))

			unmarshal := mapGetOrDefault(dec.customUnmarshalers, t.Elem(), defaultUnmarshal)
			return unmarshal(dec, v.Elem())
		}

	case reflect.Slice:
		defaultUnmarshal := defaultUnmarshaler(t.Elem())

		return func(dec *Decoder, v reflect.Value) error {
			n, err := readLen(dec)
			if err != nil {
				return err
			}
			if n == 0 {
				v.SetZero()
				return nil
			}

			if err := accountForType(dec.Budget(), t.Elem(), n); err != nil {
				return err
			}

			v.Set(reflect.MakeSlice(t, n, n))

			unmarshal := mapGetOrDefault(dec.customUnmarshalers, t.Elem(), defaultUnmarshal)
			for i := range n {
				if err := unmarshal(dec, v.Index(i)); err != nil {
					return err
				}
			}
			return nil
		}

	case reflect.String:
		return func(dec *Decoder, v reflect.Value) error {
			n, err := readLen(dec)
			if err != nil {
				return err
			}

			if err := dec.Budget().Account(n); err != nil {
				return err
			}

			buf := dec.Scratch(n)
			if err := readBytes(dec, buf); err != nil {
				return err
			}
			// TODO: is this optimization worth it?
			if v.String() != string(buf) {
				v.SetString(string(buf))
			}
			return nil
		}

	case reflect.Struct:
		fs := structInfo(t)
		if len(fs) == 0 {
			return unmarshalEmpty
		}

		return func(dec *Decoder, v reflect.Value) error {
			for _, f := range fs {
				unmarshal := mapGetOrDefault(dec.customUnmarshalers, f.Type, f.DefaultUnmarshal)
				if err := unmarshal(dec, v.Field(f.Index)); err != nil {
					return err
				}
			}
			return nil
		}

	default:
		err := fmt.Errorf("no default unmarshaler for %v", t)
		return func(dec *Decoder, v reflect.Value) error {
			return err
		}
	}
}

func unmarshalEmpty(dec *Decoder, v reflect.Value) error {
	return nil
}

func readBytes(dec *Decoder, b []byte) error {
	_, err := io.ReadFull(dec.Reader(), b)
	return err
}

func readUint(dec *Decoder, n int) (uint64, error) {
	buf := dec.Scratch(8)
	clear(buf)
	if err := readBytes(dec, buf[:n]); err != nil {
		return 0, nil
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func readBool(dec *Decoder) (bool, error) {
	x, err := readUint(dec, 1)
	if err != nil {
		return false, err
	}
	if x != 0 && x != 1 {
		return false, fmt.Errorf("bad") // TODO: nicer error message
	}
	return x != 0, nil
}

func readLen(dec *Decoder) (int, error) {
	x, err := readUint(dec, 8)
	if err != nil {
		return -1, err
	}
	if x != uint64(int(x)) {
		return -1, fmt.Errorf("bad")
	}
	if int(x) < 0 {
		return -1, fmt.Errorf("bad")
	}
	return int(x), nil
}
