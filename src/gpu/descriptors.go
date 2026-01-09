package gpu

import (
	"unsafe"

	"worldspawn/gpu/vk"
)

var resourceDescAlloc = newSlotAlloc(1e6) // TODO: allocate at runtime
var resourceDescAllocHint int64

var samplerSlots = newSlotAlloc(2e3) // TODO: allocate at runtime
var samplerAllocHint int64

// TODO: should be moved into runtime, not be visible in the base package
func BindDescriptorSet(cb vk.CommandBuffer, bindPoint vk.PipelineBindPoint) {
	vkFns.CmdBindDescriptorSets(
		cb,
		bindPoint,
		pipelineLayout,
		0,
		1, &descriptorSet,
		0, nil)
}

// TODO: kill this as soon as we can
func PushConstants(cb vk.CommandBuffer, args []byte) {
	vkFns.CmdPushConstants(
		cb,
		pipelineLayout,
		vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
		0,
		uint32(len(args)), unsafe.Pointer(unsafe.SliceData(args)))
}
