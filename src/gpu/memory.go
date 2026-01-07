package gpu

import (
	"crypto/rand"
	"fmt"
	"runtime"
	"slices"
	"structs"
	"sync"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: experiment with host memory import. That has a few problems:
//
// We need to figure out the size of memory we're importing. We could either
// accept that parameter explicitly, or recover from Go's memory management
// structures (oof.)
//
// We need to round the imported region to a multiple of a page size. This would
// mean imported regions will overlap. This means this whole thing will only
// work if we can map at the same host and device addresses, and we'll also need
// to keep the "import count" of a page.
//
// We need to unimport the memory before the host frees it, otherwise the next
// submit is going to lose the device.

// TODO: fire up a goroutine to clean up cache every now and then. Or think
// *really hard* about cache purging policy.

var allocPoolMu sync.Mutex
var allocPool = make(map[int][]*deviceMemory) // should be our own struct instead.

func roundUpDeviceAllocationSize(size int) int {
	newSize := (size + 0x10000 - 1) &^ (0x10000 - 1)
	if newSize < size {
		panic("overflow")
	}
	return newSize
}

// TODO: rename to just memory? memoryArea (linux uses vm_area)
type deviceMemory struct {
	// deviceAddr uint64
	hostAddr uintptr // TODO: rename this and deviceAddr to something else?
	size     int
	memory   vk.DeviceMemory // rename to vkDeviceMemory and the other thing to vkBuffer?
	buffer   vk.Buffer
	// TODO: (reduced) image create info for dedicated allocs.
}

// TODO: rename these structs and vars to something clearer
//
// TODO: even with device address commands, we'll still need to specify some
// bits about dst and src like whether they're in host or device domain, etc, so
// we should clean this up because we'll keep it forever.

var allocsMu sync.Mutex
var deviceAddrs []uint64
var allocs []*deviceMemory

// TODO: make the pointer types opaque?

type Pointer[T any] uint64

func (p Pointer[T]) Value() *T {
	return (*T)(UnsafePointer(p).Value())
}

type UnsafePointer uint64

// TODO: implement String for this so it prints %x

func (p UnsafePointer) Value() unsafe.Pointer {
	allocsMu.Lock()
	defer allocsMu.Unlock()

	// TODO: use a radix tree instead of binary search
	i, ok := slices.BinarySearch(deviceAddrs, uint64(p))
	if !ok {
		i--
	}
	if 0 <= i && i < len(deviceAddrs) && deviceAddrs[i] <= uint64(p) && uint64(p) < deviceAddrs[i]+uint64(allocs[i].size) {
		off := int(uint64(p) - deviceAddrs[i])
		return unsafe.Pointer(allocs[i].hostAddr + uintptr(off))
	}
	return nil
}

// TODO: change this to simply return *deviceMemory + offset as different
// callers might want different things, and rename accordingly. We'll need to
// add some methods on the *deviceMemory to handle nil case more conveniently.
//
// TODO: make this a standalone module-internal function
func bufferAndOffset(p UnsafePointer) (buffer vk.Buffer, offset vk.DeviceSize) {
	allocsMu.Lock()
	defer allocsMu.Unlock()

	i, _ := slices.BinarySearch(deviceAddrs, uint64(p))
	if i < len(deviceAddrs) && deviceAddrs[i] <= uint64(p) && uint64(p) < deviceAddrs[i]+uint64(allocs[i].size) {
		off := vk.DeviceSize(uint64(p) - deviceAddrs[i])
		return allocs[i].buffer, off
	}
	return vk.NULL_HANDLE, 0
}

// TODO: make these flags private
const (
	hostLocal    = 1 << 0 // TODO: do we need this flag?
	hostMapped   = 1 << 1 // TODO: remove/hide this flag later and have it be always-on for buffers.
	hostUncached = 1 << 2

	// TODO: a flag for descriptor buffer
)

// TODO: see if we can/should pass unsafe.Pointer for hostAddr
// TODO:
func malloc(size int, flags uint32) UnsafePointer {
	// TODO: handle size=0
	if size <= 0 {
		panic("bad")
	}
	if size > 1<<60 {
		panic("too big")
	}

	// TODO: actually round size up to the page size at least.

	// TODO: fast small allocator and cache of large allocations

	size = roundUpDeviceAllocationSize(size)

	var pinner runtime.Pinner
	defer pinner.Unpin()

	gpuInit()

	bufferCreateInfo := pinned(&pinner, &vk.BufferCreateInfo{
		SType: vk.STRUCTURE_TYPE_BUFFER_CREATE_INFO,
		Size:  vk.DeviceSize(size),
		// TODO: use BufferUsageFlags2CreateInfoKHR when it's core
		// TODO: convert our flags into vk flags
		Usage: vk.BufferUsageFlags(
			vk.BUFFER_USAGE_TRANSFER_SRC_BIT | vk.BUFFER_USAGE_TRANSFER_DST_BIT |
				vk.BUFFER_USAGE_STORAGE_BUFFER_BIT |
				vk.BUFFER_USAGE_INDEX_BUFFER_BIT |
				vk.BUFFER_USAGE_SHADER_BINDING_TABLE_BIT_KHR |
				vk.BUFFER_USAGE_SHADER_DEVICE_ADDRESS_BIT |
				vk.BUFFER_USAGE_ACCELERATION_STRUCTURE_BUILD_INPUT_READ_ONLY_BIT_KHR |
				vk.BUFFER_USAGE_ACCELERATION_STRUCTURE_STORAGE_BIT_KHR),
		SharingMode: vk.SHARING_MODE_EXCLUSIVE, // TODO: should be concurrent if we're using more queue families
	})

	requirements := &vk.MemoryRequirements2{
		SType: vk.STRUCTURE_TYPE_MEMORY_REQUIREMENTS_2,
	}
	vkFns.GetDeviceBufferMemoryRequirements(device, &vk.DeviceBufferMemoryRequirements{
		SType:       vk.STRUCTURE_TYPE_DEVICE_BUFFER_MEMORY_REQUIREMENTS,
		PCreateInfo: bufferCreateInfo,
	}, requirements)

	if requirements.Size != vk.DeviceSize(size) {
		panic("bug")
	}

	memoryTypeIndex := findMemoryTypeIndex(requirements.MemoryTypeBits, flags)

	memoryAllocateInfo := &vk.MemoryAllocateInfo{
		SType: vk.STRUCTURE_TYPE_MEMORY_ALLOCATE_INFO,
		PNext: unsafe.Pointer(pinned(&pinner, &vk.MemoryAllocateFlagsInfo{
			SType: vk.STRUCTURE_TYPE_MEMORY_ALLOCATE_FLAGS_INFO,
			Flags: vk.MemoryAllocateFlags(vk.MEMORY_ALLOCATE_DEVICE_ADDRESS_BIT),
		})),
		AllocationSize:  vk.DeviceSize(size),
		MemoryTypeIndex: uint32(memoryTypeIndex),
	}
	var memory vk.DeviceMemory
	must(vkFns.AllocateMemory(device, memoryAllocateInfo, nil, &memory))

	var buffer vk.Buffer
	must(vkFns.CreateBuffer(device, bufferCreateInfo, nil, &buffer))

	must(vkFns.BindBufferMemory2(device, 1, &vk.BindBufferMemoryInfo{
		SType:        vk.STRUCTURE_TYPE_BIND_BUFFER_MEMORY_INFO,
		Buffer:       buffer,
		Memory:       memory,
		MemoryOffset: 0,
	}))

	deviceAddr := uint64(vkFns.GetBufferDeviceAddress(device, &vk.BufferDeviceAddressInfo{
		SType:  vk.STRUCTURE_TYPE_BUFFER_DEVICE_ADDRESS_INFO,
		Buffer: buffer,
	}))

	var hostAddr uintptr
	if flags&hostMapped != 0 {
		must(vkFns.MapMemory(device, memory, 0, vk.DeviceSize(size), 0, (*unsafe.Pointer)(unsafe.Pointer(&hostAddr))))

		if false {
			// TODO: make this opt-in. Or alternatively just make sure we zero
			// things out.
			hehe := unsafe.Slice((*byte)(unsafe.Pointer(hostAddr)), size)
			rand.Read(hehe)
		}
	}

	allocsMu.Lock()
	i, _ := slices.BinarySearch(deviceAddrs, deviceAddr)
	deviceAddrs = slices.Insert(deviceAddrs, i, deviceAddr)
	allocs = slices.Insert(allocs, i, &deviceMemory{
		hostAddr: hostAddr,
		size:     size,
		memory:   memory,
		buffer:   buffer,
	})
	allocsMu.Unlock()

	return UnsafePointer(deviceAddr)
}

// TODO: when poking the cache, we should just repeatedly probe buckets of
// increasing sizes probably

func NewUncached[T any]() Pointer[T] {
	return (Pointer[T])(malloc(int(unsafe.Sizeof(*new(T))), hostMapped /*|hostUncached*/))
}

// TODO: remove in favor of just host []T eventually
type Slice[T any] struct {
	_    structs.HostLayout
	data Pointer[T]
	len  int
	cap  int
}

func MakeSliceUncached[T any](n int) Slice[T] {
	return Slice[T]{
		data: Pointer[T](malloc(int(unsafe.Sizeof(*new(T)))*n, hostMapped /*|hostUncached*/)),
		len:  n,
		cap:  n,
	}
}

func (s Slice[T]) Index(i int) Pointer[T] {
	return Pointer[T](uint64(s.data) + uint64(i*int(unsafe.Sizeof(*new(T)))))
}

func (s Slice[T]) Value() []T {
	return unsafe.Slice((*T)(UnsafePointer(s.data).Value()), s.cap)[:s.len]
}

func SliceData[T any](s Slice[T]) Pointer[T] {
	return s.data
}

func SliceLen[T any](s Slice[T]) int {
	return s.len
}

// TODO: replace with pointer any and also teach this to free slices.
func Free(pointer UnsafePointer) {
	if pointer == 0 {
		return
	}

	allocsMu.Lock()
	defer allocsMu.Unlock()

	// If it was allocated by the big allocator, just throw it into the cache
	// and arm the timer that will eventually clean this up.

	i, _ := slices.BinarySearch(deviceAddrs, uint64(pointer))
	if deviceAddrs[i] == uint64(pointer) {
		vkFns.DestroyBuffer(device, allocs[i].buffer, nil)
		vkFns.FreeMemory(device, allocs[i].memory, nil)

		deviceAddrs = slices.Delete(deviceAddrs, i, i+1)
		allocs = slices.Delete(allocs, i, i+1)

		return
	}

	panic(fmt.Sprintf("gpu: attempting to free %x which was not allocated", pointer))
}

// TODO: make this testable by passing memProps as an argument
func findMemoryTypeIndex(memoryTypeBits uint32, flags uint32) int {
	memProps := &vk.PhysicalDeviceMemoryProperties2{
		SType: vk.STRUCTURE_TYPE_PHYSICAL_DEVICE_MEMORY_PROPERTIES_2,
	}
	vkFns.GetPhysicalDeviceMemoryProperties2(physicalDevice, memProps)

	var try []vk.MemoryPropertyFlags
	switch flags &^ hostLocal {
	case 0:
		try = []vk.MemoryPropertyFlags{
			vk.MemoryPropertyFlags(vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT),
			0,
		}
	case hostMapped:
		try = []vk.MemoryPropertyFlags{
			vk.MemoryPropertyFlags(vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT | vk.MEMORY_PROPERTY_HOST_COHERENT_BIT | vk.MEMORY_PROPERTY_HOST_CACHED_BIT),
			vk.MemoryPropertyFlags(vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT | vk.MEMORY_PROPERTY_HOST_COHERENT_BIT),
		}
	case hostMapped | hostUncached:
		if flags&hostLocal != 0 {
			try = []vk.MemoryPropertyFlags{
				vk.MemoryPropertyFlags(vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT | vk.MEMORY_PROPERTY_HOST_COHERENT_BIT),
			}
		} else {
			try = []vk.MemoryPropertyFlags{
				vk.MemoryPropertyFlags(vk.MEMORY_PROPERTY_DEVICE_LOCAL_BIT | vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT | vk.MEMORY_PROPERTY_HOST_COHERENT_BIT),
				vk.MemoryPropertyFlags(vk.MEMORY_PROPERTY_HOST_VISIBLE_BIT | vk.MEMORY_PROPERTY_HOST_COHERENT_BIT),
			}
		}
	default:
		panic("bad choice of flags")
	}

	for _, wantProps := range try {
		for i, memType := range memProps.MemoryTypes[:memProps.MemoryTypeCount] {
			if memoryTypeBits&(1<<i) != 0 && memType.PropertyFlags&wantProps == wantProps {
				return i
			}
		}
	}
	return -1
}
