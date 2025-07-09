package gpu

import (
	"fmt"
	"sync"

	"worldspawn/gpu/vk"
)

type commandBuffer struct {
	vk     vk.CommandBuffer
	memory vk.CommandPool
}

func newCommandBuffer(queueFamily uint32) *commandBuffer {
	var vkCommandPool vk.CommandPool
	if err := vkFns.CreateCommandPool(device, &vk.CommandPoolCreateInfo{
		SType: vk.STRUCTURE_TYPE_COMMAND_POOL_CREATE_INFO,
		// TODO: I don't think we need either of these flags. remove?
		Flags:            vk.CommandPoolCreateFlags(vk.COMMAND_POOL_CREATE_TRANSIENT_BIT | vk.COMMAND_POOL_CREATE_RESET_COMMAND_BUFFER_BIT),
		QueueFamilyIndex: queueFamily,
	}, nil, &vkCommandPool); err != nil {
		panic(fmt.Sprintf("gpu: vkCreateCommandPool: %v", err))
	}

	var vkCommandBuffer vk.CommandBuffer
	if err := vkFns.AllocateCommandBuffers(device, &vk.CommandBufferAllocateInfo{
		SType:              vk.STRUCTURE_TYPE_COMMAND_BUFFER_ALLOCATE_INFO,
		CommandPool:        vkCommandPool,
		Level:              vk.COMMAND_BUFFER_LEVEL_PRIMARY,
		CommandBufferCount: 1,
	}, &vkCommandBuffer); err != nil {
		panic(fmt.Sprintf("gpu: vkAllocateCommandBuffers: %v", err))
	}

	return &commandBuffer{
		vk:     vkCommandBuffer,
		memory: vkCommandPool,
	}
}

func (cb *commandBuffer) Vk() vk.CommandBuffer {
	return cb.vk
}

// TODO: move this to another file?
// TODO: I still don't like how we do cache
type commandBufferCache struct {
	queueFamily uint32

	mu    sync.Mutex
	cache []*commandBuffer
}

// TODO: move the definition into gpu.go
// TODO: initialize this in gpuInit instead?
var cbcaches [32]commandBufferCache

func init() {
	for i := range cbcaches {
		cbcaches[i].queueFamily = uint32(i)
	}
}

func (cache *commandBufferCache) Get() *commandBuffer {
	// TODO: clean this up
	cache.mu.Lock()
	var cb *commandBuffer
	if len(cache.cache) > 0 {
		cb = cache.cache[len(cache.cache)-1]
		cache.cache = cache.cache[:len(cache.cache)-1]
	}
	cache.mu.Unlock()
	if cb == nil {
		cb = newCommandBuffer(cache.queueFamily)
	}
	return cb
}

func (cache *commandBufferCache) Put(cb *commandBuffer) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.cache = append(cache.cache, cb)
}
