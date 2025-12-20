package nice2

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strconv"
)

type NoDefaultArshalerError struct{ Type reflect.Type }

func (e NoDefaultArshalerError) Error() string {
	return fmt.Sprintf("no default arshaler for %v", e.Type)
}

func DefaultArshalers(t reflect.Type, arshalers Arshalers) Arshaler {
	switch t.Kind() {
	case reflect.Bool:
		return Arshaler{
			Size: 1,
			Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
				var x byte
				if v.Bool() {
					x = 1
				}
				heap.Object(p)[0] = x
				return nil
			},
			Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
				x := heap.Object(p)[0]
				if x != 0 && x != 1 {
					// TODO: better error message
					return errors.New("bad bool")
				}
				v.SetBool(x != 0)
				return nil
			},
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n := int(t.Size())
		if t.Kind() == reflect.Int {
			n = 8
		}
		return Arshaler{
			Size: n,
			Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
				MarshalUint(heap, p, uint64(v.Int()), n*8)
				return nil
			},
			Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
				v.SetInt(int64(UnmarshalUint(heap, p, n*8)))
				return nil
			},
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n := int(t.Size())
		if t.Kind() == reflect.Uint {
			n = 8
		}
		return Arshaler{
			Size: n,
			Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
				MarshalUint(heap, p, v.Uint(), n*8)
				return nil
			},
			Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
				v.SetUint(UnmarshalUint(heap, p, n*8))
				return nil
			},
		}

	case reflect.Float32:
		return Arshaler{
			Size: 4,
			Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
				MarshalUint(heap, p, uint64(math.Float32bits(float32(v.Float()))), 32)
				return nil
			},
			Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
				v.SetFloat(float64(math.Float32frombits(uint32(UnmarshalUint(heap, p, 32)))))
				return nil
			},
		}

	case reflect.Float64:
		return Arshaler{
			Size: 8,
			Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
				MarshalUint(heap, p, math.Float64bits(v.Float()), 64)
				return nil
			},
			Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
				v.SetFloat(math.Float64frombits(UnmarshalUint(heap, p, 64)))
				return nil
			},
		}

	case reflect.Array:
		len := t.Len()
		elemArshaler := arshalers.Get(t.Elem())

		return Arshaler{
			Size: len * elemArshaler.Size,
			Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
				for i := range len {
					if err := elemArshaler.Marshal(heap, p.Add(i*elemArshaler.Size), v.Index(i)); err != nil {
						return err
					}
				}
				return nil
			},
			Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
				for i := range len {
					if err := elemArshaler.Unmarshal(heap, p.Add(i*elemArshaler.Size), v.Index(i)); err != nil {
						return err
					}
				}
				return nil
			},
		}

	case reflect.Map:
		return Arshaler{
			Size: 16,
			Marshal: func(heap Heap, p Pointer, m reflect.Value) error {
				if m.Len() == 0 {
					MarshalPointer(heap, p.Add(0), Pointer{-1})
					marshalLen(heap, p.Add(8), 0)
					return nil
				}

				keyArshaler := arshalers.Get(t.Key())
				valueArshaler := arshalers.Get(t.Elem())
				stride := keyArshaler.Size + valueArshaler.Size

				dataP := heap.New(m.Len() * stride)
				// if dataAddr == -1

				MarshalPointer(heap, p.Add(0), dataP)
				marshalLen(heap, p.Add(8), m.Len())
				key := reflect.New(t.Key()).Elem()
				value := reflect.New(t.Elem()).Elem()
				for iter := m.MapRange(); iter.Next(); {
					key.SetIterKey(iter)
					value.SetIterValue(iter)
					if err := keyArshaler.Marshal(heap, dataP, key); err != nil {
						return err
					}
					if err := valueArshaler.Marshal(heap, dataP.Add(keyArshaler.Size), value); err != nil {
						return err
					}
					dataP.Add(stride)
				}
				return nil
			},
			Unmarshal: func(heap Heap, p Pointer, m reflect.Value) error {
				dataP := UnmarshalPointer(heap, p.Add(0))
				len := unmarshalLen(heap, p.Add(8))
				if len == 0 {
					if dataP != (Pointer{-1}) {
						return errors.New("empty map with non-nil data")
					}
					m.SetZero()
					return nil
				}

				keyArshaler := arshalers.Get(t.Key())
				valueArshaler := arshalers.Get(t.Elem())
				stride := keyArshaler.Size + valueArshaler.Size
				if err := validate(heap, dataP, len*stride); err != nil {
					return err
				}

				m.Set(reflect.MakeMapWithSize(t, len))
				key := reflect.New(t.Key()).Elem()
				value := reflect.New(t.Elem()).Elem()
				for range len {
					if err := keyArshaler.Unmarshal(heap, dataP, key); err != nil {
						return err
					}
					if err := valueArshaler.Unmarshal(heap, dataP.Add(keyArshaler.Size), value); err != nil {
						return err
					}
					m.SetMapIndex(key, value)
					dataP.Add(stride)
				}
				return nil
			},
		}

	case reflect.Pointer:
		return Arshaler{
			Size: 8,
			Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
				if v.IsNil() {
					MarshalPointer(heap, p, Pointer{-1})
					return nil
				}

				elemArshaler := arshalers.Get(t.Elem())

				elemP := heap.New(elemArshaler.Size)

				MarshalPointer(heap, p, elemP)
				return elemArshaler.Marshal(heap, elemP, v.Elem())
			},
			Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
				elemP := UnmarshalPointer(heap, p)
				if elemP == (Pointer{-1}) {
					v.SetZero()
					return nil
				}

				elemArshaler := arshalers.Get(t.Elem())
				if err := validate(heap, elemP, elemArshaler.Size); err != nil {
					return err
				}

				v.Set(reflect.New(t.Elem()))
				return elemArshaler.Unmarshal(heap, elemP, v.Elem())
			},
		}

	case reflect.Slice:
		return Arshaler{
			Size: 16,
			Marshal: func(heap Heap, p Pointer, s reflect.Value) error {
				if s.Len() == 0 {
					MarshalPointer(heap, p.Add(0), Pointer{-1})
					marshalLen(heap, p.Add(8), 0)
					return nil
				}

				elemArshaler := arshalers.Get(t.Elem())

				dataP := heap.New(s.Len() * elemArshaler.Size)
				// if dataAddr == -1 {}

				MarshalPointer(heap, p.Add(0), dataP)
				marshalLen(heap, p.Add(8), s.Len())
				for i := range s.Len() {
					if err := elemArshaler.Marshal(heap, dataP.Add(i*elemArshaler.Size), s.Index(i)); err != nil {
						return ArshalingError{"[" + strconv.Itoa(i) + "]", err}
					}
				}
				return nil
			},
			Unmarshal: func(heap Heap, p Pointer, s reflect.Value) error {
				dataP := UnmarshalPointer(heap, p.Add(0))
				len := unmarshalLen(heap, p.Add(8))
				if len == 0 {
					if dataP != (Pointer{-1}) {
						return errors.New("empty slice with non-nil data")
					}
					s.SetZero()
					return nil
				}

				elemArshaler := arshalers.Get(t.Elem())
				if err := validate(heap, dataP, len*elemArshaler.Size); err != nil {
					return err
				}

				s.Set(reflect.MakeSlice(t, len, len))
				for i := range len {
					if err := elemArshaler.Unmarshal(heap, dataP.Add(i*elemArshaler.Size), s.Index(i)); err != nil {
						return ArshalingError{"[" + strconv.Itoa(i) + "]", err}
					}
				}
				return nil
			},
		}

	case reflect.String:
		return Arshaler{
			Size: 16,
			Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
				if v.Len() == 0 {
					MarshalPointer(heap, p.Add(0), Pointer{-1})
					marshalLen(heap, p.Add(8), 0)
					return nil
				}

				dataP := heap.New(v.Len())

				MarshalPointer(heap, p.Add(0), dataP)
				marshalLen(heap, p.Add(8), v.Len())
				copy(heap.Object(dataP), v.String())
				return nil
			},
			Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
				dataP := UnmarshalPointer(heap, p.Add(0))
				len := unmarshalLen(heap, p.Add(8))
				if len == 0 {
					if dataP != (Pointer{-1}) {
						return errors.New("empty string with non-nil data")
					}
					v.SetZero()
					return nil
				}

				if err := validate(heap, dataP, len); err != nil {
					return err
				}

				v.SetString(string(heap.Object(dataP)[:len]))
				return nil
			},
		}

	case reflect.Struct:
		// type field struct {
		// 	index int
		// 	off   int
		// }
		type fieldArshaler struct {
			// TODO: include field index
			off       int
			marshal   func(heap Heap, p Pointer, v reflect.Value) error
			unmarshal func(heap Heap, p Pointer, v reflect.Value) error
		}
		// TODO: deinterleave into 3 (field index x offset, marshaler,
		// unmarshaler)?
		fieldArshalers := make([]fieldArshaler, t.NumField())
		size := 0
		for i := range t.NumField() {
			f := t.Field(i)

			arshaler := arshalers.Get(f.Type)

			fieldArshalers[i] = fieldArshaler{
				off:       size,
				marshal:   arshaler.Marshal,
				unmarshal: arshaler.Unmarshal,
			}
			size += arshaler.Size
		}

		return Arshaler{
			Size: size,
			Marshal: func(heap Heap, addr Pointer, v reflect.Value) error {
				for i, fieldArshaler := range fieldArshalers {
					if err := fieldArshaler.marshal(heap, addr.Add(fieldArshaler.off), v.Field(i)); err != nil {
						return ArshalingError{"." + t.Field(i).Name, err}
					}
				}
				return nil
			},
			Unmarshal: func(heap Heap, addr Pointer, v reflect.Value) error {
				for i, fieldArshaler := range fieldArshalers {
					if err := fieldArshaler.unmarshal(heap, addr.Add(fieldArshaler.off), v.Field(i)); err != nil {
						return ArshalingError{"." + t.Field(i).Name, err}
					}
				}
				return nil
			},
		}

	default:
		err := NoDefaultArshalerError{t}

		return Arshaler{
			Marshal:   func(heap Heap, addr Pointer, v reflect.Value) error { return err },
			Unmarshal: func(heap Heap, addr Pointer, v reflect.Value) error { return err },
		}
	}
}

// TODO: allow for error in these {,un}marshalers?

func UnmarshalPointer(heap Heap, p Pointer) Pointer {
	off := int(UnmarshalUint(heap, p, 64))
	if off == 0 {
		return Pointer{-1}
	}
	return p.Add(off)
}

func unmarshalLen(heap Heap, p Pointer) int {
	return int(UnmarshalUint(heap, p, 64))
}

func UnmarshalUint(heap Heap, p Pointer, n int) uint64 {
	b := heap.Object(p)
	switch n {
	case 8:
		return uint64(b[0])
	case 16:
		return uint64(binary.LittleEndian.Uint16(b))
	case 32:
		return uint64(binary.LittleEndian.Uint32(b))
	case 64:
		return binary.LittleEndian.Uint64(b)
	default:
		panic("unreachable")
	}
}

func MarshalPointer(heap Heap, p, q Pointer) {
	if q.addr == -1 {
		MarshalUint(heap, p, 0, 64)
		return
	}
	MarshalUint(heap, p, uint64(q.addr-p.addr), 64)
}

func marshalLen(heap Heap, p Pointer, i int) {
	MarshalUint(heap, p, uint64(i), 64)
}

func MarshalUint(heap Heap, p Pointer, v uint64, n int) {
	b := heap.Object(p)
	switch n {
	case 8:
		b[0] = byte(v)
	case 16:
		binary.LittleEndian.PutUint16(b, uint16(v))
	case 32:
		binary.LittleEndian.PutUint32(b, uint32(v))
	case 64:
		binary.LittleEndian.PutUint64(b, v)
	default:
		panic("unreachable")
	}
}
