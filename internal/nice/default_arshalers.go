package nice

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"sync"
)

// TODO: respect types implementing encoding.BinaryAppender and
// encoding.BinaryMarshaler and their respective unmarshaling parts?

// TODO: encoding/binary is currently faster than us in some cases. We should
// analyze why and see where we can improve.

// TODO: explore optimizing un/marshaling of certain structs with a memcpy

const optimizedArshalers = false

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

var emptyStructArshaler = arshaler{
	marshal:   func(enc *Encoder, v reflect.Value) error { return nil },
	unmarshal: func(dec *Decoder, v reflect.Value) error { return nil },
}

func makeDefaultArshaler(t reflect.Type) arshaler {
	switch t.Kind() {
	case reflect.Bool:
		return arshaler{
			marshal: func(enc *Encoder, v reflect.Value) error {
				return marshalBool(enc, v.Bool())
			},
			unmarshal: func(dec *Decoder, v reflect.Value) error {
				b, err := unmarshalBool(dec)
				if err == nil {
					v.SetBool(b)
				}
				return err
			},
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := int(t.Size())
		if t.Kind() == reflect.Int {
			n = 8
		}
		return arshaler{
			marshal: func(enc *Encoder, v reflect.Value) error {
				return marshalUint(enc, uint64(v.Int()), n)
			},
			unmarshal: func(dec *Decoder, v reflect.Value) error {
				x, err := unmarshalUint(dec, n)
				if err == nil {
					v.SetInt(int64(x))
				}
				return err
			},
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n := int(t.Size())
		if t.Kind() == reflect.Uint {
			n = 8
		}
		return arshaler{
			marshal: func(enc *Encoder, v reflect.Value) error {
				return marshalUint(enc, v.Uint(), n)
			},
			unmarshal: func(dec *Decoder, v reflect.Value) error {
				x, err := unmarshalUint(dec, n)
				if err == nil {
					v.SetUint(x)
				}
				return err
			},
		}

	case reflect.Float32:
		return arshaler{
			marshal: func(enc *Encoder, v reflect.Value) error {
				x := uint64(math.Float32bits(float32(v.Float())))
				return marshalUint(enc, x, 4)
			},
			unmarshal: func(dec *Decoder, v reflect.Value) error {
				x, err := unmarshalUint(dec, 4)
				if err == nil {
					v.SetFloat(float64(math.Float32frombits(uint32(x))))
				}
				return err
			},
		}

	case reflect.Float64:
		return arshaler{
			marshal: func(enc *Encoder, v reflect.Value) error {
				x := math.Float64bits(v.Float())
				return marshalUint(enc, x, 8)
			},
			unmarshal: func(dec *Decoder, v reflect.Value) error {
				x, err := unmarshalUint(dec, 8)
				if err == nil {
					v.SetFloat(math.Float64frombits(x))
				}
				return err
			},
		}

	case reflect.Array:
		n := t.Len()
		if n == 0 {
			return emptyStructArshaler
		}

		return arshaler{
			marshal: func(enc *Encoder, v reflect.Value) error {
				marshal := enc.getArshaler(t.Elem()).marshal
				for i := range n {
					if err := marshal(enc, v.Index(i)); err != nil {
						return ArshalingError{fmt.Sprintf("[%#v]", i), err}
					}
				}
				return nil
			},
			unmarshal: func(dec *Decoder, v reflect.Value) error {
				unmarshal := dec.getArshaler(t.Elem()).unmarshal
				for i := range n {
					if err := unmarshal(dec, v.Index(i)); err != nil {
						return ArshalingError{fmt.Sprintf("[%#v]", i), err}
					}
				}
				return nil
			},
		}

	case reflect.Map:
		return arshaler{
			marshal: func(enc *Encoder, m reflect.Value) error {
				n := m.Len()
				if err := marshalLen(enc, n); err != nil {
					return err
				}
				if n == 0 {
					return nil
				}

				marshalKey := enc.getArshaler(t.Key()).marshal
				marshalVal := enc.getArshaler(t.Elem()).marshal

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
						return ArshalingError{fmt.Sprintf("[%#v]", k.Interface()), err}
					}
				}
				return nil
			},
			unmarshal: func(dec *Decoder, m reflect.Value) error {
				n, err := unmarshalLen(dec)
				if err != nil {
					return err
				}
				if n == 0 {
					m.SetZero()
					return nil
				}

				if err := budgetDrawMap(dec.Budget(), t, n); err != nil {
					return err
				}

				m.Set(reflect.MakeMapWithSize(t, n))

				unmarshalKey := dec.getArshaler(t.Key()).unmarshal
				unmarshalVal := dec.getArshaler(t.Elem()).unmarshal

				k := reflect.New(t.Key()).Elem()
				v := reflect.New(t.Elem()).Elem()
				for range n {
					if err := unmarshalKey(dec, k); err != nil {
						return err
					}
					if err := unmarshalVal(dec, v); err != nil {
						return ArshalingError{fmt.Sprintf("[%#v]", k.Interface()), err}
					}
					m.SetMapIndex(k, v)
				}
				return nil
			},
		}

	case reflect.Pointer:
		// TODO: we could have a field option to assume the field is never nil
		// so it's marshaled trivially
		return arshaler{
			marshal: func(enc *Encoder, v reflect.Value) error {
				isNonNil := !v.IsNil()
				if err := marshalBool(enc, isNonNil); err != nil {
					return err
				}
				if !isNonNil {
					return nil
				}

				marshal := enc.getArshaler(t.Elem()).marshal
				return marshal(enc, v.Elem())
			},
			unmarshal: func(dec *Decoder, v reflect.Value) error {
				isNonNil, err := unmarshalBool(dec)
				if err != nil {
					return err
				}
				if !isNonNil {
					v.SetZero()
					return nil
				}

				if err := budgetDrawN(dec.Budget(), t.Elem(), 1); err != nil {
					return err
				}

				v.Set(reflect.New(t.Elem()))

				unmarshal := dec.getArshaler(t.Elem()).unmarshal
				return unmarshal(dec, v.Elem())
			},
		}

	case reflect.Slice:
		// TODO: unify with array marshaler to the extent possible
		return arshaler{
			marshal: func(enc *Encoder, v reflect.Value) error {
				n := v.Len()
				if err := marshalLen(enc, n); err != nil {
					return err
				}
				if n == 0 {
					return nil
				}

				// TODO: enforce size limits when encoding too

				marshal := enc.getArshaler(t.Elem()).marshal
				for i := range n {
					if err := marshal(enc, v.Index(i)); err != nil {
						return ArshalingError{fmt.Sprintf("[%#v]", i), err}
					}
				}
				return nil
			},
			unmarshal: func(dec *Decoder, v reflect.Value) error {
				n, err := unmarshalLen(dec)
				if err != nil {
					return err
				}
				if n == 0 {
					v.SetZero()
					return nil
				}

				if err := budgetDrawN(dec.Budget(), t.Elem(), n); err != nil {
					return err
				}

				v.Set(reflect.MakeSlice(t, n, n))

				unmarshal := dec.getArshaler(t.Elem()).unmarshal
				for i := range n {
					if err := unmarshal(dec, v.Index(i)); err != nil {
						return ArshalingError{fmt.Sprintf("[%#v]", i), err}
					}
				}
				return nil
			},
		}

	case reflect.String:
		return arshaler{
			marshal: func(enc *Encoder, v reflect.Value) error {
				// TODO: optimize for non-StringWriters as well?
				// TODO: see if there's anything to do to optimize small string
				// writes
				s := v.String()
				// TODO: enforce size limit
				if err := marshalLen(enc, len(s)); err != nil {
					return err
				}
				if _, err := io.WriteString(enc.Writer(), s); err != nil {
					return err
				}
				return nil
			},
			unmarshal: func(dec *Decoder, v reflect.Value) error {
				n, err := unmarshalLen(dec)
				if err != nil {
					return err
				}

				if err := dec.Budget().Draw(n); err != nil {
					return err
				}

				buf := dec.Scratch(n)
				if err := decodeBytes(dec, buf); err != nil {
					return err
				}
				// TODO: is this optimization worth it?
				if v.String() != string(buf) {
					v.SetString(string(buf))
				}
				return nil
			},
		}

	case reflect.Struct:
		return makeDefaultStructArshaler(t)

	default:
		// TODO: make a type for this?
		err := fmt.Errorf("no default arshaler for %v", t)
		return arshaler{
			marshal:   func(enc *Encoder, v reflect.Value) error { return err },
			unmarshal: func(dec *Decoder, v reflect.Value) error { return err },
		}
	}
}

// TODO: rename
type structField struct {
	index int
	typ   reflect.Type
}

// TODO: rename
func structFields(t reflect.Type) []structField {
	var fields []structField
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		// TODO: handle embedded fields
		fields = append(fields, structField{
			index: i,
			typ:   f.Type,
		})
	}
	return fields
}

func makeDefaultStructArshaler(t reflect.Type) arshaler {
	fields := structFields(t)
	if len(fields) == 0 {
		return emptyStructArshaler
	}

	return arshaler{
		marshal: func(enc *Encoder, v reflect.Value) error {
			for _, f := range fields {
				marshal := enc.getArshaler(f.typ).marshal
				if err := marshal(enc, v.Field(f.index)); err != nil {
					return ArshalingError{"." + t.Field(f.index).Name, err}
				}
			}
			return nil
		},
		unmarshal: func(dec *Decoder, v reflect.Value) error {
			for _, f := range fields {
				unmarshal := dec.getArshaler(f.typ).unmarshal
				if err := unmarshal(dec, v.Field(f.index)); err != nil {
					return ArshalingError{"." + t.Field(f.index).Name, err}
				}
			}
			return nil
		},
	}
}

func marshalBool(enc *Encoder, x bool) error {
	b := byte(0)
	if x {
		b = 1
	}
	enc.Scratch = append(enc.Scratch[:0], b)
	return encodeBytes(enc, enc.Scratch)
}

func unmarshalBool(dec *Decoder) (bool, error) {
	buf := dec.Scratch(1)
	if err := decodeBytes(dec, buf); err != nil {
		return false, err
	}
	b := buf[0]
	if b != 0 && b != 1 {
		return false, fmt.Errorf("bad") // TODO: nicer error message
	}
	return b != 0, nil
}

// TODO: change n to be in number of bits?
func marshalUint(enc *Encoder, x uint64, n int) error {
	enc.Scratch = binary.LittleEndian.AppendUint64(enc.Scratch[:0], x)
	return encodeBytes(enc, enc.Scratch[:n])
}

func unmarshalUint(dec *Decoder, n int) (uint64, error) {
	buf := dec.Scratch(8)
	clear(buf)
	if err := decodeBytes(dec, buf[:n]); err != nil {
		return 0, nil
	}
	return binary.LittleEndian.Uint64(buf), nil
}

// TODO: let the user specify whether ints (and thus lengths) should be
// marshaled as 32 or 64 bits? Or we could encode ints as varints but that's
// ass.
func marshalLen(enc *Encoder, x int) error {
	if x < 0 {
		panic("bad")
	}
	return marshalUint(enc, uint64(x), 8)
}

func unmarshalLen(dec *Decoder) (int, error) {
	x, err := unmarshalUint(dec, 8)
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
