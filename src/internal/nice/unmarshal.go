package nice

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"sync"
)

func UnmarshalDecode(dec *Decoder, out any) error {
	p := reflect.ValueOf(out)
	if p.Kind() != reflect.Pointer || p.IsNil() {
		panic("expecting a non-nil pointer")
	}

	v := p.Elem()

	t := v.Type()

	unmarshal := loadOrElse(dec.customUnmarshalers, t, func() unmarshaler { return defaultUnmarshaler(t) })
	return unmarshal(dec, v)
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
			buf := dec.Scratch(8)
			if err := readBytes(dec, buf[:n]); err != nil {
				return err
			}
			v.SetInt(int64(binary.LittleEndian.Uint64(buf)))
			return nil
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n := int(t.Size())
		if t.Kind() == reflect.Uint {
			n = 8
		}
		return func(dec *Decoder, v reflect.Value) error {
			buf := dec.Scratch(8)
			if err := readBytes(dec, buf[:n]); err != nil {
				return err
			}
			v.SetUint(binary.LittleEndian.Uint64(buf))
			return nil
		}

	case reflect.Float32:
		return func(dec *Decoder, v reflect.Value) error {
			buf, err := readBytes2(dec, 4)
			if err != nil {
				return err
			}
			v.SetFloat(float64(math.Float32frombits(binary.LittleEndian.Uint32(buf))))
			return nil
		}

	case reflect.Float64:
		return func(dec *Decoder, v reflect.Value) error {
			buf, err := readBytes2(dec, 8)
			if err != nil {
				return err
			}
			v.SetFloat(math.Float64frombits(binary.LittleEndian.Uint64(buf)))
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
			n, err := readInt(dec)
			if err != nil {
				return err
			}
			if n == 0 {
				m.SetZero()
				return nil
			}

			if err := accountT(dec.Budget(), t.Key(), n); err != nil {
				return err
			}
			if err := accountT(dec.Budget(), t.Elem(), n); err != nil {
				return err
			}

			m.Set(reflect.MakeMapWithSize(t, n))

			unmarshalKey := loadOrDefault(dec.customUnmarshalers, t.Key(), defaultUnmarshalKey)
			unmarshalVal := loadOrDefault(dec.customUnmarshalers, t.Elem(), defaultUnmarshalVal)

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

			if err := accountT(dec.Budget(), t.Elem(), 1); err != nil {
				return err
			}

			v.Set(reflect.New(t.Elem()))

			unmarshal := loadOrDefault(dec.customUnmarshalers, t.Elem(), defaultUnmarshal)
			return unmarshal(dec, v.Elem())
		}

	case reflect.Slice:
		defaultUnmarshal := defaultUnmarshaler(t.Elem())

		return func(dec *Decoder, v reflect.Value) error {
			n, err := readInt(dec)
			if err != nil {
				return err
			}
			if n == 0 {
				v.SetZero()
				return nil
			}

			if err := accountT(dec.Budget(), t.Elem(), n); err != nil {
				return err
			}

			v.Set(reflect.MakeSlice(t, n, n))

			unmarshal := loadOrDefault(dec.customUnmarshalers, t.Elem(), defaultUnmarshal)
			for i := range n {
				if err := unmarshal(dec, v.Index(i)); err != nil {
					return err
				}
			}
			return nil
		}

	case reflect.String:
		return func(dec *Decoder, v reflect.Value) error {
			n, err := readInt(dec)
			if err != nil {
				return err
			}

			if err := dec.Budget().Account(n); err != nil {
				return err
			}

			buf, err := readBytes2(dec, n)
			if err != nil {
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
				unmarshal := loadOrDefault(dec.customUnmarshalers, f.Type, f.DefaultUnmarshal)
				if err := unmarshal(dec, v.Field(f.Index)); err != nil {
					return err
				}
			}
			return nil
		}
	}

	return func(dec *Decoder, v reflect.Value) error {
		return fmt.Errorf("no default unmarshaler for %v", t)
	}
}

func unmarshalEmpty(dec *Decoder, v reflect.Value) error {
	return nil
}

// TODO: distinguish readBytes and readBytes2 better

func readBytes(dec *Decoder, b []byte) error {
	_, err := io.ReadFull(dec.Reader(), b)
	return err
}

func readBytes2(dec *Decoder, n int) ([]byte, error) {
	buf := dec.Scratch(n)
	if err := readBytes(dec, buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func readBool(dec *Decoder) (bool, error) {
	buf, err := readBytes2(dec, 1)
	if err != nil {
		return false, err
	}
	if buf[0] > 1 {
		return false, fmt.Errorf("bad") // TODO: nicer error message
	}
	return buf[0] != 0, nil
}

func readInt(dec *Decoder) (int, error) {
	buf, err := readBytes2(dec, 8)
	if err != nil {
		return 0, nil
	}
	x := int(binary.LittleEndian.Uint64(buf))
	if x < 0 {
		return 0, fmt.Errorf("bad")
	}
	return int(x), nil
}
