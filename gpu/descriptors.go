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

// TODO: the hints should be per-thread or similar

// TODO: we'll need a public api for allocating resource and sampler
// descriptors, getting a pointer to their bytes, etc. We could do that by
// introducing a type ResourceDescriptor int and a similar SamplerDescriptor
// with a pile of methods that use resourceHeap underneath. That would allow us
// to avoid exposing the heap objects directly, as well as the hints.

// TODO: rename this to something else
type descriptorHeap struct {
	elemSize int
	memory   UnsafePointer
	alloc    bitslice.BitSlice
	len      int // TODO: we could remove it and use alloc.Cap() instead
	// TODO: reserved range stuff
}

func (heap *descriptorHeap) init(elemSize int, len int) {
	*heap = descriptorHeap{}
	heap.elemSize = elemSize
	heap.memory = malloc(len*elemSize, hostMapped|mallocDescriptorHeap)
	heap.alloc = bitslice.Make(len)
	heap.alloc.Swap(0, true)
	// TODO: Set the reserved heap range to true here
	heap.len = len
}

func (heap *descriptorHeap) Alloc(hintp *int64) int {
	hint := atomic.LoadInt64(hintp)

	i := int(hint)
	if i == 0 {
		// Hint was uninitialized. Choose the start index at random in hopes
		// that we won't contend with others.
		i = rand.IntN(heap.alloc.Len())
	}
	// Round down the start index to the bitset's word boundary.
	i = i / 64 * 64

	i = heap.alloc.FindAndSet(i)
	if i < 0 {
		// Try again, starting at 0.
		i = heap.alloc.FindAndSet(0)
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
	return i
}

func (heap *descriptorHeap) Free(i int) {
	if i == 0 {
		return
	}
	if heap.alloc.Swap(i, false) != true {
		panic(fmt.Sprintf("tried to free slot %d that was not allocated", i))
	}
}

// TODO: panic when the user tries to map 0th, reserved or unallocated
// descriptor
func (heap *descriptorHeap) Map(i int) Slice[byte] {
	if i == 0 {
		panic("trying to map a reserved descriptor")
	}
	return SliceAt(Pointer[byte](UnsafePointerAdd(heap.memory, i*heap.elemSize)), heap.elemSize)
}

var (
	resourceDescAllocHint int64
	resourceHeap          descriptorHeap

	samplerAllocHint int64
	samplerHeap      descriptorHeap
)

type ResourceDescriptor int

func (rd ResourceDescriptor) Map() Slice[byte] {
	return resourceHeap.Map(int(rd))
}

// TODO: should be moved into runtime, not be visible in the base package
func BindDescriptorHeaps(cb vk.CommandBuffer) {
	for index, heap := range []*descriptorHeap{&samplerHeap, &resourceHeap} {
		info := &vk.BindHeapInfoEXT{
			SType: vk.STRUCTURE_TYPE_BIND_HEAP_INFO_EXT,
			HeapRange: vk.DeviceAddressRangeKHR{
				Address: vk.DeviceAddress(heap.memory),
				Size:    vk.DeviceSize(heap.elemSize * heap.len),
			},
		}
		switch index {
		case 0:
			vkFns.CmdBindSamplerHeapEXT(cb, info)
		case 1:
			vkFns.CmdBindResourceHeapEXT(cb, info)
		}
	}
}
