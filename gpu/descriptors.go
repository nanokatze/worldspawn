package gpu

import (
	"fmt"
	"math/rand/v2"
	"sync/atomic"

	"worldspawn/gpu/internal/bitslice"
	"worldspawn/gpu/vk"
)

// TODO: we can't actually allocate heaps lazily, so we have to create them at
// init time. I guess we'll provide the user with the ways to configure/override
// their sizes prior to gpu init, similar to fd table, so that if the user's app
// doesn't use the heaps it can avoid paying the cost for them, which can help
// if the app's footprint is otherwise tiny.

// TODO: we'll need a public api for allocating resource and sampler
// descriptors, getting a pointer to their bytes, etc. We could do that by
// introducing a type ResourceDescriptor int and a similar SamplerDescriptor
// with a pile of methods that use resourceHeap underneath. That would allow us
// to avoid exposing the heap objects directly, as well as the hints.

// TODO: place the reserved range at the end of the allocation instead, that way
// we can avoid exposing it to the user and also avoid any special treatment in
// the allocator by simply managing the shorter range

type descriptorHeap struct {
	elemSize     int
	memory       Slice[byte]
	alloc        bitslice.BitSlice
	reservedSize int // TODO: this wouldn't be necessary if we were to place the reserved range at the end
}

func (heap *descriptorHeap) init(elemSize int, len int, reservedSize int) {
	*heap = descriptorHeap{}
	heap.elemSize = elemSize
	heap.memory = SliceAt(Pointer[byte](malloc(len, hostMapped|mallocDescriptorHeap)), len)
	heap.alloc = bitslice.Make(len / elemSize)
	// always reserve 0; TODO: rethink this decision
	heap.alloc.Swap(0, true)
	for i := 0; i < reservedSize; i += elemSize {
		heap.alloc.Swap(i/elemSize, true)
	}
	heap.reservedSize = reservedSize
}

func (heap *descriptorHeap) Alloc(bytes int, hintp *int64) int {
	if bytes%heap.elemSize != 0 {
		panic("not aligned")
	}
	n := bytes / heap.elemSize

	hint := atomic.LoadInt64(hintp)

	i := int(hint)
	if i == 0 {
		// Hint was uninitialized. Choose the start index at random in hopes
		// that we won't contend with others.
		i = rand.IntN(heap.alloc.Len())
	}
	// Round down the start index to the bitset's word boundary.
	i = i / 64 * 64

	i = heap.alloc.FindAndSet(i, n)
	if i < 0 {
		// Try again, starting at 0.
		i = heap.alloc.FindAndSet(0, n)
	}

	if i < 0 {
		panic("out of free slots")
	}
	if i == 0 {
		panic("unreachable")
	}

	// Ok

	if hint != int64(i) {
		// We succeeded and picked an index different from the hint. Try to
		// update the hint, but don't bother if someone already has done so,
		// to avoid cache line ping-pong.
		atomic.CompareAndSwapInt64(hintp, hint, int64(i))
	}
	return i * heap.elemSize
}

func (heap *descriptorHeap) Free(offset, bytes int) {
	if offset%heap.elemSize != 0 {
		panic("not aligned")
	}
	if bytes%heap.elemSize != 0 {
		panic("not aligned")
	}
	i := offset / heap.elemSize
	if i == 0 {
		return
	}
	for n := range bytes / heap.elemSize {
		if heap.alloc.Swap(i+n, false) != true {
			panic(fmt.Sprintf("tried to free slot %d that was not allocated", i))
		}
	}
}

func (heap *descriptorHeap) Base() Slice[byte] {
	return heap.memory
}

var (
	resourceHeap descriptorHeap
	samplerHeap  descriptorHeap
)

// TODO: should be moved into runtime, not be visible in the base package
func BindDescriptorHeaps(cb vk.CommandBuffer) {
	for index, heap := range []*descriptorHeap{&samplerHeap, &resourceHeap} {
		info := &vk.BindHeapInfoEXT{
			SType: vk.STRUCTURE_TYPE_BIND_HEAP_INFO_EXT,
			HeapRange: vk.DeviceAddressRangeKHR{
				Address: vk.DeviceAddress(SliceData(heap.memory)),
				Size:    vk.DeviceSize(SliceLen(heap.memory)),
			},
			ReservedRangeOffset: 0,
			ReservedRangeSize:   vk.DeviceSize(heap.reservedSize),
		}
		switch index {
		case 0:
			VkFns.CmdBindSamplerHeapEXT(cb, info)
		case 1:
			VkFns.CmdBindResourceHeapEXT(cb, info)
		}
	}
}
