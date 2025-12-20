package nice2

import (
	"fmt"
	"maps"
	"reflect"
)

type arshalerMap struct {
	m map[reflect.Type]Arshalers
	f Arshalers
}

func WithInterfaceArshaler[T any](types []reflect.Type) func(m *arshalerMap) {
	t := reflect.TypeFor[T]()

	m1 := maps.Collect(func(yield func(reflect.Type, uint64) bool) {
		for idx, typ := range types {
			yield(typ, uint64(idx))
		}
	})
	m2 := maps.Collect(func(yield func(uint64, reflect.Type) bool) {
		for idx, typ := range types {
			yield(uint64(idx), typ)
		}
	})

	// TODO: we need to support nil
	arshaler := func(_ reflect.Type, arshalers Arshalers) Arshaler {
		return Arshaler{
			Size: 16,
			Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
				elem := v.Elem()
				elemType := elem.Type()

				typeId, ok := m1[elemType]
				if !ok {
					panic(fmt.Sprintf("bad %#v", v))
				}

				elemArshaler := arshalers.Get(elemType)

				elemP := heap.New(elemArshaler.Size)

				MarshalUint(heap, p.Add(0), typeId, 64)
				MarshalPointer(heap, p.Add(8), elemP)

				// TODO: any way we could avoid an alloc?
				tmp := reflect.New(elemType).Elem()
				tmp.Set(elem)
				return elemArshaler.Marshal(heap, elemP, tmp)
			},
			Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
				typeId := UnmarshalUint(heap, p.Add(0), 64)
				elemP := UnmarshalPointer(heap, p.Add(8))

				elemType, ok := m2[typeId]
				if !ok {
					return fmt.Errorf("unknown input command")
				}

				elemArshaler := arshalers.Get(elemType)

				// TODO: any way we could avoid an alloc?
				tmp := reflect.New(elemType).Elem()
				if err := elemArshaler.Unmarshal(heap, elemP, tmp); err != nil {
					return err
				}
				v.Set(tmp)
				return nil
			},
		}
	}

	return func(m *arshalerMap) { m.m[t] = arshaler }
}

func MakeArshalerMap(opts ...func(m *arshalerMap)) Arshalers {
	m := &arshalerMap{
		m: make(map[reflect.Type]Arshalers),
		f: DefaultArshalers,
	}
	for _, opt := range opts {
		opt(m)
	}
	return makeArshalerCache(m.Get)
}

func (m arshalerMap) Get(t reflect.Type, arshalers Arshalers) Arshaler {
	f, ok := m.m[t]
	if !ok {
		f = m.f
	}
	return f(t, arshalers)
}
