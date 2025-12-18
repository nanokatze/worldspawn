package nice2

import (
	"encoding/binary"
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

// TODO: remove the heap prefix?

// TODO: allow for error
func HeapReadPtr(heap Heap, p Pointer) Pointer {
	off := int(HeapReadUint(heap, p, 64))
	if off == 0 {
		return Pointer{-1}
	}
	return p.Add(off)
}

// TODO: allow for error?
func heapReadLen(heap Heap, p Pointer) int {
	return int(HeapReadUint(heap, p, 64))
}

func HeapReadUint(heap Heap, p Pointer, n int) uint64 {
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

func HeapWritePtr(heap Heap, p, q Pointer) {
	if q.addr == -1 {
		HeapWriteUint(heap, p, 0, 64)
		return
	}
	HeapWriteUint(heap, p, uint64(q.addr-p.addr), 64)
}

func heapWriteLen(heap Heap, p Pointer, i int) {
	HeapWriteUint(heap, p, uint64(i), 64)
}

func HeapWriteUint(heap Heap, p Pointer, v uint64, n int) {
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
