package nice2

import (
	"reflect"
)

func Marshal(in any, arshalers Arshalers) ([]byte, error) {
	p := reflect.ValueOf(in)
	// check that p is a pointer
	v := p.Elem()
	t := v.Type()

	arshaler := arshalers.Get(t)

	heap := InMemoryHeap{}
	heap.buf = make([]byte, arshaler.Size)
	err := arshaler.Marshal(&heap, startOfHeap, v)

	return heap.buf, err
}

func Marshal2(dst []byte, in any, arshalers Arshalers) ([]byte, error) {
	p := reflect.ValueOf(in)
	// check that p is a pointer
	v := p.Elem()
	t := v.Type()

	arshaler := arshalers.Get(t)

	heap := InMemoryHeap{}
	heap.buf = append(dst[:0], make([]byte, arshaler.Size)...)
	err := arshaler.Marshal(&heap, startOfHeap, v)

	return heap.buf, err
}

func Unmarshal(b []byte, in any, arshalers Arshalers) error {
	p := reflect.ValueOf(in)
	// check that p is a pointer
	v := p.Elem()
	t := v.Type()

	arshaler := arshalers.Get(t)

	heap := &InMemoryHeap{b}
	if err := validate(heap, startOfHeap, arshaler.Size); err != nil {
		return err
	}
	return arshaler.Unmarshal(heap, startOfHeap, v)
}
