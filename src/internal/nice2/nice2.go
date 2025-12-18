package nice2

import (
	"reflect"
)

// TODO: keep track of where we are so we can produce very good quality
// diagnostics?

/*
const (
	_ = iota
	accessLinkIndex
	accessLinkKey
	accessLinkField
)

type accessChain struct {
	// chain []
}
*/

type ArshalerGetter func(t reflect.Type) Arshaler

func Marshal(in any, getArshaler ArshalerGetter) ([]byte, error) {
	p := reflect.ValueOf(in)
	// check that p is a pointer
	v := p.Elem()
	t := v.Type()

	heap := InMemoryHeap{}
	heap.buf = make([]byte, getArshaler(t).Size)
	err := getArshaler(t).Marshal(&heap, startOfHeap, v)

	return heap.buf, err
}

func Unmarshal(b []byte, in any, getArshaler ArshalerGetter) error {
	p := reflect.ValueOf(in)
	// check that p is a pointer
	v := p.Elem()
	t := v.Type()

	heap := &InMemoryHeap{b}
	if err := validate(heap, startOfHeap, getArshaler(t).Size); err != nil {
		return err
	}
	return getArshaler(t).Unmarshal(heap, startOfHeap, v)
}
