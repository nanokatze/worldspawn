package nice2

import (
	"errors"
)

// TODO: make it be non-0
var startOfHeap = Pointer{0}

type Heap interface {
	New(size int) Pointer

	// The returned slice is valid until the next call to New.
	Object(p Pointer) []byte
}

// TODO: better representation of a nil pointer
type Pointer struct{ addr int }

func (p Pointer) Add(off int) Pointer { return Pointer{p.addr + off} }

// TODO: size limit?
type InMemoryHeap struct {
	buf []byte
}

// TODO: rename?
func (h *InMemoryHeap) New(size int) Pointer {
	// TODO: align
	addr := len(h.buf)
	// TODO: allocate without zeroing
	h.buf = append(h.buf, make([]byte, size)...)
	return Pointer{addr}
}

// TODO: rename?
func (h *InMemoryHeap) Object(p Pointer) []byte {
	return h.buf[p.addr:]
}

/*
type flushableHeap interface {
	Flush(p Pointer) error
}

// TODO: do we really need to return errors here?
func heapFlush(heap Heap, p Pointer) error {
	if flushable, ok := heap.(flushableHeap); ok {
		return flushable.Flush(p)
	}
	return nil
}
*/

func validate(heap Heap, p Pointer, size int) error {
	if len(heap.Object(p)) < size {
		return errors.New("out of heap")
	}
	return nil
}
