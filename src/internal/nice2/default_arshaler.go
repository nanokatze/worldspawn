package nice2

import (
	"errors"
	"fmt"
	"math"
	"reflect"
)

type NoDefaultArshalerError struct{ Type reflect.Type }

func (e NoDefaultArshalerError) Error() string {
	return fmt.Sprintf("no default arshaler for %v", e.Type)
}

// TODO: have an option for 32-bit pointers and lengths? That would actually
// have to be communicated through getArshaler.
// TODO: make public?
//
// Returns the default arshaler for the type t. The arshalers used for
// serializing member types are fetched using the provided getArshaler.
func defaultArshalerGetter(getArshaler ArshalerGetter) ArshalerGetter {
	return func(t reflect.Type) Arshaler {
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
					HeapWriteUint(heap, p, uint64(v.Int()), n*8)
					return nil
				},
				Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
					v.SetInt(int64(HeapReadUint(heap, p, n*8)))
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
					HeapWriteUint(heap, p, v.Uint(), n*8)
					return nil
				},
				Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
					v.SetUint(HeapReadUint(heap, p, n*8))
					return nil
				},
			}

		case reflect.Float32:
			return Arshaler{
				Size: 4,
				Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
					HeapWriteUint(heap, p, uint64(math.Float32bits(float32(v.Float()))), 32)
					return nil
				},
				Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
					v.SetFloat(float64(math.Float32frombits(uint32(HeapReadUint(heap, p, 32)))))
					return nil
				},
			}

		case reflect.Float64:
			return Arshaler{
				Size: 8,
				Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
					HeapWriteUint(heap, p, math.Float64bits(v.Float()), 64)
					return nil
				},
				Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
					v.SetFloat(math.Float64frombits(HeapReadUint(heap, p, 64)))
					return nil
				},
			}

		case reflect.Array:
			len := t.Len()
			arshaler := getArshaler(t.Elem())

			return Arshaler{
				Size: len * arshaler.Size,
				Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
					for i := range len {
						if err := arshaler.Marshal(heap, p.Add(i*arshaler.Size), v.Index(i)); err != nil {
							return err
						}
					}
					return nil
				},
				Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
					for i := range len {
						if err := arshaler.Unmarshal(heap, p.Add(i*arshaler.Size), v.Index(i)); err != nil {
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
						HeapWritePtr(heap, p.Add(0), Pointer{-1})
						heapWriteLen(heap, p.Add(8), 0)
						return nil
					}

					keyArshaler := getArshaler(t.Key())
					valArshaler := getArshaler(t.Elem())

					stride := keyArshaler.Size + valArshaler.Size

					data := heap.New(m.Len() * stride)
					// if dataAddr == -1

					HeapWritePtr(heap, p.Add(0), data)
					heapWriteLen(heap, p.Add(8), m.Len())

					k := reflect.New(t.Key()).Elem()
					v := reflect.New(t.Elem()).Elem()
					for iter := m.MapRange(); iter.Next(); {
						k.SetIterKey(iter)
						v.SetIterValue(iter)
						if err := keyArshaler.Marshal(heap, data, k); err != nil {
							return err
						}
						if err := valArshaler.Marshal(heap, data.Add(keyArshaler.Size), v); err != nil {
							return err
						}
						data.Add(stride)
					}
					return nil
				},
				Unmarshal: func(heap Heap, p Pointer, m reflect.Value) error {
					data := HeapReadPtr(heap, p.Add(0))
					len := heapReadLen(heap, p.Add(8))
					if len == 0 {
						if data != (Pointer{-1}) {
							return errors.New("empty map with non-nil data")
						}
						m.SetZero()
						return nil
					}

					keyArshaler := getArshaler(t.Key())
					valArshaler := getArshaler(t.Elem())

					stride := keyArshaler.Size + valArshaler.Size

					if err := validate(heap, data, len*stride); err != nil {
						return err
					}
					m.Set(reflect.MakeMapWithSize(t, len))
					k := reflect.New(t.Key()).Elem()
					v := reflect.New(t.Elem()).Elem()
					for range len {
						if err := keyArshaler.Unmarshal(heap, data, k); err != nil {
							return err
						}
						if err := valArshaler.Unmarshal(heap, data.Add(keyArshaler.Size), v); err != nil {
							return err
						}
						m.SetMapIndex(k, v)
						data.Add(stride)
					}
					return nil
				},
			}

		case reflect.Pointer:
			return Arshaler{
				Size: 8,
				Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
					if v.IsNil() {
						HeapWritePtr(heap, p, Pointer{-1})
						return nil
					}
					arshaler := getArshaler(t.Elem())
					data := heap.New(arshaler.Size)
					HeapWritePtr(heap, p, data)
					return arshaler.Marshal(heap, data, v.Elem())
				},
				Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
					data := HeapReadPtr(heap, p)
					if data == (Pointer{-1}) {
						v.SetZero()
						return nil
					}

					arshaler := getArshaler(t.Elem())
					if err := validate(heap, data, arshaler.Size); err != nil {
						return err
					}
					v.Set(reflect.New(t.Elem()))
					return arshaler.Unmarshal(heap, data, v.Elem())
				},
			}

		case reflect.Slice:
			return Arshaler{
				Size: 16,
				Marshal: func(heap Heap, p Pointer, s reflect.Value) error {
					if s.Len() == 0 {
						HeapWritePtr(heap, p.Add(0), Pointer{-1})
						heapWriteLen(heap, p.Add(8), 0)
						return nil
					}

					arshaler := getArshaler(t.Elem())

					data := heap.New(s.Len() * arshaler.Size)
					// if dataAddr == -1 {}

					HeapWritePtr(heap, p.Add(0), data)
					heapWriteLen(heap, p.Add(8), s.Len())

					for i := range s.Len() {
						if err := arshaler.Marshal(heap, data.Add(i*arshaler.Size), s.Index(i)); err != nil {
							return err
						}
					}
					return nil
				},
				Unmarshal: func(heap Heap, p Pointer, s reflect.Value) error {
					data := HeapReadPtr(heap, p.Add(0))
					len := heapReadLen(heap, p.Add(8))
					if len == 0 {
						if data != (Pointer{-1}) {
							return errors.New("empty slice with non-nil data")
						}
						s.SetZero()
						return nil
					}

					arshaler := getArshaler(t.Elem())
					if err := validate(heap, data, len*arshaler.Size); err != nil {
						return err
					}
					s.Set(reflect.MakeSlice(t, len, len))
					for i := range len {
						if err := arshaler.Unmarshal(heap, data.Add(i*arshaler.Size), s.Index(i)); err != nil {
							return err
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
						HeapWritePtr(heap, p.Add(0), Pointer{-1})
						heapWriteLen(heap, p.Add(8), 0)
						return nil
					}

					data := heap.New(v.Len())

					HeapWritePtr(heap, p.Add(0), data)
					heapWriteLen(heap, p.Add(8), v.Len())

					copy(heap.Object(data), v.String())
					return nil
				},
				Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
					data := HeapReadPtr(heap, p.Add(0))
					len := heapReadLen(heap, p.Add(8))
					if len == 0 {
						if data != (Pointer{-1}) {
							return errors.New("empty string with non-nil data")
						}
						v.SetZero()
						return nil
					}
					if err := validate(heap, data, len); err != nil {
						return err
					}
					v.SetString(string(heap.Object(data)[:len]))
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

				arshaler := getArshaler(f.Type)

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
					for i, arshaler := range fieldArshalers {
						if err := arshaler.marshal(heap, addr.Add(arshaler.off), v.Field(i)); err != nil {
							return err
						}
					}
					return nil
				},
				Unmarshal: func(heap Heap, addr Pointer, v reflect.Value) error {
					for i, arshaler := range fieldArshalers {
						if err := arshaler.unmarshal(heap, addr.Add(arshaler.off), v.Field(i)); err != nil {
							return err
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
}
