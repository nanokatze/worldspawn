package nice2

import (
	"reflect"
)

type Arshaler struct {
	Size int
	// TODO: rename to Encode, Decode?
	Marshal   func(heap Heap, p Pointer, v reflect.Value) error
	Unmarshal func(heap Heap, p Pointer, v reflect.Value) error
}

type TypedArshaler[T any] struct {
	Size      int
	Marshal   func(heap Heap, p Pointer, v *T) error
	Unmarshal func(heap Heap, p Pointer, v *T) error
}

func (typedArshaler TypedArshaler[T]) Arshaler() Arshaler {
	return Arshaler{
		Size: typedArshaler.Size,
		Marshal: func(heap Heap, p Pointer, v reflect.Value) error {
			return typedArshaler.Marshal(heap, p, v.Addr().Interface().(*T))
		},
		Unmarshal: func(heap Heap, p Pointer, v reflect.Value) error {
			return typedArshaler.Unmarshal(heap, p, v.Addr().Interface().(*T))
		},
	}
}
