package nice2

import (
	"reflect"
)

type Arshaler struct {
	Size      int
	Marshal   func(heap Heap, p Pointer, v reflect.Value) error
	Unmarshal func(heap Heap, p Pointer, v reflect.Value) error
}

type Arshalers func(t reflect.Type, arshalers Arshalers) Arshaler

func (arshalers Arshalers) Get(t reflect.Type) Arshaler { return arshalers(t, arshalers) }
