package nice2

import (
	"fmt"
	"reflect"
)

// TODO: make ID generic
func InterfaceArshaler[T any](m map[reflect.Type]uint64, m2 map[uint64]reflect.Type) func(getArshaler ArshalerGetter) TypedArshaler[T] {
	return func(getArshaler ArshalerGetter) TypedArshaler[T] {
		return TypedArshaler[T]{
			Size: 16,
			Marshal: func(heap Heap, p Pointer, x *T) error {
				data := reflect.ValueOf(*x)
				typ := data.Type()

				id, ok := m[typ]
				if !ok {
					panic(fmt.Sprintf("bad %#v", *x))
				}

				arshaler := getArshaler(typ)

				datap := heap.New(arshaler.Size)

				HeapWriteUint(heap, p.Add(0), id, 64)
				HeapWritePtr(heap, p.Add(8), datap)

				tmp := reflect.New(typ).Elem()
				tmp.Set(data)
				return arshaler.Marshal(heap, datap, tmp)
			},
			Unmarshal: func(heap Heap, p Pointer, x *T) error {
				id := HeapReadUint(heap, p.Add(0), 64)
				datap := HeapReadPtr(heap, p.Add(8))

				typ, ok := m2[id]
				if !ok {
					return fmt.Errorf("unknown input command")
				}

				arshaler := getArshaler(typ)

				// TODO: any way we could avoid an alloc?
				data := reflect.New(typ).Elem()
				if err := arshaler.Unmarshal(heap, datap, data); err != nil {
					return err
				}
				*x, _ = reflect.TypeAssert[T](data)
				return nil
			},
		}
	}
}
