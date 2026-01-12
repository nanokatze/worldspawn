package vk

// TODO: make mkfuncs.go generate this

/*
#cgo LDFLAGS: -lvulkan

#include <stddef.h>
#include <stdint.h>

#include <vulkan/vulkan_core.h>

// TODO: these must be vulkan call or whatever, i.e. use stdcall on windows.

// TODO: change prefix to govkcall? _cgovkcall? ...

uint32_t
vkcall_32_uintptr(uint32_t (*fn)(uintptr_t), uintptr_t a0)
{
	return fn(a0);
}

uint32_t
vkcall_32_uintptr_32_ptr(uint32_t (*fn)(uintptr_t, uint32_t, void*), uintptr_t a0, uint32_t a1, void *a2)
{
	return fn(a0, a1, a2);
}

uint32_t
vkcall_32_uintptr_32_ptr_32_64(uint32_t (*fn)(uintptr_t, uint32_t, void*, uint32_t, uint64_t), uintptr_t a0, uint32_t a1, void *a2, uint32_t a3, uint64_t a4)
{
	return fn(a0, a1, a2, a3, a4);
}

uint32_t
vkcall_32_uintptr_32_ptr_64(uint32_t (*fn)(uintptr_t, uint32_t, void*, uint64_t), uintptr_t a0, uint32_t a1, void *a2, uint64_t a3)
{
	return fn(a0, a1, a2, a3);
}

uint32_t
vkcall_32_uintptr_32_ptr_ptr(uint32_t (*fn)(uintptr_t, uint32_t, void*, void*), uintptr_t a0, uint32_t a1, void *a2, void *a3)
{
	return fn(a0, a1, a2, a3);
}

uint32_t
vkcall_32_uintptr_32_ptr_ptr_ptr(uint32_t (*fn)(uintptr_t, uint32_t, void*, void*, void*), uintptr_t a0, uint32_t a1, void *a2, void *a3, void *a4)
{
	return fn(a0, a1, a2, a3, a4);
}

uint32_t
vkcall_32_uintptr_64_32_32_uintptr_ptr(uint32_t (*fn)(uintptr_t, uint64_t, uint32_t, uint32_t, uintptr_t, void*), uintptr_t a0, uint64_t a1, uint32_t a2, uint32_t a3, uintptr_t a4, void *a5)
{
	return fn(a0, a1, a2, a3, a4, a5);
}

uint32_t
vkcall_32_uintptr_64_64_32_ptr_ptr_ptr(uint32_t (*fn)(uintptr_t, uint64_t, uint64_t, uint32_t, void*, void*, void*), uintptr_t a0, uint64_t a1, uint64_t a2, uint32_t a3, void *a4, void *a5, void *a6)
{
	return fn(a0, a1, a2, a3, a4, a5, a6);
}

uint32_t
vkcall_32_uintptr_64_ptr_ptr(uint32_t (*fn)(uintptr_t, uint64_t, void*, void*), uintptr_t a0, uint64_t a1, void *a2, void *a3)
{
	return fn(a0, a1, a2, a3);
}

uint32_t
vkcall_32_uintptr_ptr(uint32_t (*fn)(uintptr_t, void*), uintptr_t a0, void *a1)
{
	return fn(a0, a1);
}

#cgo noescape vkcall_32_uintptr_ptr_64_noescape_nocallback
#cgo nocallback vkcall_32_uintptr_ptr_64_noescape_nocallback
uint32_t
vkcall_32_uintptr_ptr_64_noescape_nocallback(uint32_t (*fn)(uintptr_t, void*, uint64_t), uintptr_t a0, void *a1, uint64_t a2)
{
	return fn(a0, a1, a2);
}

uint32_t
vkcall_32_uintptr_ptr_ptr(uint32_t (*fn)(uintptr_t, void*, void*), uintptr_t a0, void *a1, void *a2)
{
	return fn(a0, a1, a2);
}

#cgo noescape vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback
#cgo nocallback vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback
uint32_t
vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(uint32_t (*fn)(uintptr_t, void*, void*, void*), uintptr_t a0, void *a1, void *a2, void *a3)
{
	return fn(a0, a1, a2, a3);
}

uint64_t
vkcall_64_uintptr_ptr(uint64_t (*fn)(uintptr_t, void*), uintptr_t a0, void *a1)
{
	return fn(a0, a1);
}

uint64_t
vkcall_64_uintptr_64_32_32(uint64_t (*fn)(uintptr_t, uint64_t, uint32_t, uint32_t), uintptr_t a0, uint64_t a1, uint32_t a2, uint32_t a3)
{
	return fn(a0, a1, a2, a3);
}

void
vkcall_void_uintptr(void (*fn)(uintptr_t), uintptr_t a0)
{
	fn(a0);
}

void
vkcall_void_uintptr_32(void (*fn)(uintptr_t, uint32_t), uintptr_t a0, uint32_t a1)
{
	fn(a0, a1);
}

void
vkcall_void_uintptr_32_32_32(void (*fn)(uintptr_t, uint32_t, uint32_t, uint32_t), uintptr_t a0, uint32_t a1, uint32_t a2, uint32_t a3)
{
	fn(a0, a1, a2, a3);
}

void
vkcall_void_uintptr_32_32_32_32(void (*fn)(uintptr_t, uint32_t, uint32_t, uint32_t, uint32_t), uintptr_t a0, uint32_t a1, uint32_t a2, uint32_t a3, uint32_t a4)
{
	fn(a0, a1, a2, a3, a4);
}

void
vkcall_void_uintptr_32_32_32_32_32(void (*fn)(uintptr_t, uint32_t, uint32_t, uint32_t, uint32_t, uint32_t), uintptr_t a0, uint32_t a1, uint32_t a2, uint32_t a3, uint32_t a4, uint32_t a5)
{
	fn(a0, a1, a2, a3, a4, a5);
}

void
vkcall_void_uintptr_32_32_ptr(void (*fn)(uintptr_t, uint32_t, uint32_t, void*), uintptr_t a0, uint32_t a1, uint32_t a2, void *a3)
{
	fn(a0, a1, a2, a3);
}

void
vkcall_void_uintptr_32_64(void (*fn)(uintptr_t, uint32_t, uint64_t),  uintptr_t a0, uint32_t a1, uint64_t a2)
{
	fn(a0, a1, a2);
}

void
vkcall_void_uintptr_32_64_32_32_ptr_32_ptr(void (*fn)(uintptr_t, uint32_t, uint64_t, uint32_t, uint32_t, void*, uint32_t, void*),  uintptr_t a0, uint32_t a1, uint64_t a2, uint32_t a3, uint32_t a4, void *a5, uint32_t a6, void *a7)
{
	fn(a0, a1, a2, a3, a4, a5, a6, a7);
}

void
vkcall_void_uintptr_32_ptr(void (*fn)(uintptr_t, uint32_t, void*), uintptr_t a0, uint32_t a1, void *a2)
{
	fn(a0, a1, a2);
}

#cgo noescape vkcall_void_uintptr_32_ptr_32_ptr_noescape_nocallback
#cgo nocallback vkcall_void_uintptr_32_ptr_32_ptr_noescape_nocallback
void
vkcall_void_uintptr_32_ptr_32_ptr_noescape_nocallback(void (*fn)(uintptr_t, uint32_t, void*, uint32_t, void*), uintptr_t a0, uint32_t a1, void *a2, uint32_t a3, void *a4)
{
	fn(a0, a1, a2, a3, a4);
}

void
vkcall_void_uintptr_32_ptr_ptr(void (*fn)(uintptr_t, uint32_t, void*, void*), uintptr_t a0, uint32_t a1, void *a2, void *a3)
{
	fn(a0, a1, a2, a3);
}

void
vkcall_void_uintptr_32_ptr_ptr_ptr(void (*fn)(uintptr_t, uint32_t, void*, void*, void*), uintptr_t a0, uint32_t a1, void *a2, void *a3, void *a4)
{
	fn(a0, a1, a2, a3, a4);
}

void
vkcall_void_uintptr_64_32_32_32_ptr(void (*fn)(uintptr_t, uint64_t, uint32_t, uint32_t, uint32_t, void*), uintptr_t a0, uint64_t a1, uint32_t a2, uint32_t a3, uint32_t a4, void *a5)
{
	fn(a0, a1, a2, a3, a4, a5);
}

void
vkcall_void_uintptr_64_64_32(void (*fn)(uintptr_t, uint64_t, uint64_t, uint32_t), uintptr_t a0, uint64_t a1, uint64_t a2, uint32_t a3)
{
	fn(a0, a1, a2, a3);
}

void
vkcall_void_uintptr_64_64_64_32(void (*fn)(uintptr_t, uint64_t, uint64_t, uint64_t, uint32_t), uintptr_t a0, uint64_t a1, uint64_t a2, uint64_t a3, uint32_t a4)
{
	fn(a0, a1, a2, a3, a4);
}

void
vkcall_void_uintptr_64_ptr(void (*fn)(uintptr_t, uint64_t, void*), uintptr_t a0, uint64_t a1, void *a2)
{
	fn(a0, a1, a2);
}

void
vkcall_void_uintptr_ptr(void (*fn)(uintptr_t, void*), uintptr_t a0, void *a1)
{
	fn(a0, a1);
}

void
vkcall_void_uintptr_ptr_ptr(void (*fn)(uintptr_t, void*, void*), uintptr_t a0, void *a1, void *a2)
{
	fn(a0, a1, a2);
}

#cgo noescape vkcall_void_uintptr_ptr_ptr_ptr_ptr_32_32_32_noescape_nocallback
#cgo nocallback vkcall_void_uintptr_ptr_ptr_ptr_ptr_32_32_32_noescape_nocallback
void
vkcall_void_uintptr_ptr_ptr_ptr_ptr_32_32_32_noescape_nocallback(void (*fn)(uintptr_t, void*, void*, void*, void*, uint32_t, uint32_t, uint32_t), uintptr_t a0, void *a1, void *a2, void *a3, void *a4, uint32_t a5, uint32_t a6, uint32_t a7)
{
	fn(a0, a1, a2, a3, a4, a5, a6, a7);
}
*/
import "C"

import (
	"reflect"
	"unsafe"
)

type InstanceFuncs struct {
}

// TODO: overall we should do stuff like volk and ash do, which is, most things
// should be called through e.g. vkGetDeviceProcAddr etc, with exceptions of
// vkCreateInstance and vkEnumerateInstanceLayerProperties

func EnumerateInstanceLayerProperties(pPropertyCount *uint32, pProperties *LayerProperties) error {
	return resultErr(Result(C.vkEnumerateInstanceLayerProperties((*C.uint32_t)(pPropertyCount), (*C.VkLayerProperties)(unsafe.Pointer(pProperties)))))
}

func CreateInstance(pCreateInfo *InstanceCreateInfo, pAllocator *AllocationCallbacks, instance *Instance) error {
	return resultErr(Result(C.vkCreateInstance((*C.VkInstanceCreateInfo)(unsafe.Pointer(pCreateInfo)), (*C.VkAllocationCallbacks)(unsafe.Pointer(pAllocator)), (*C.VkInstance)(unsafe.Pointer(instance)))))
}

// TODO: pass vkGetInstanceProcAddr
func (funcs *InstanceFuncs) Init(instance Instance) {
}

func (funcs *InstanceFuncs) CreateDevice(physicalDevice PhysicalDevice, pCreateInfo *DeviceCreateInfo, pDevice *Device) error {
	return resultErr(Result(C.vkCreateDevice(transmute[PhysicalDevice, C.VkPhysicalDevice](physicalDevice), (*C.VkDeviceCreateInfo)(unsafe.Pointer(pCreateInfo)), nil, transmute[*Device, *C.VkDevice](pDevice))))
}

func (funcs *InstanceFuncs) EnumeratePhysicalDevices(instance Instance, pPhysicalDeviceCount *uint32, pPhysicalDevices *PhysicalDevice) error {
	return resultErr(Result(C.vkEnumeratePhysicalDevices(transmute[Instance, C.VkInstance](instance), (*C.uint32_t)(pPhysicalDeviceCount), (*C.VkPhysicalDevice)(unsafe.Pointer(pPhysicalDevices)))))
}

func (funcs *InstanceFuncs) GetPhysicalDeviceFeatures2(physicalDevice PhysicalDevice, pFeatures *PhysicalDeviceFeatures2) {
	C.vkGetPhysicalDeviceFeatures2(transmute[PhysicalDevice, C.VkPhysicalDevice](physicalDevice), (*C.VkPhysicalDeviceFeatures2)(unsafe.Pointer(pFeatures)))
}

func (funcs *InstanceFuncs) GetPhysicalDeviceQueueFamilyProperties2(physicalDevice PhysicalDevice, pQueueFamilyPropertyCount *uint32, pQueueFamilyProperties *QueueFamilyProperties2) {
	C.vkGetPhysicalDeviceQueueFamilyProperties2(transmute[PhysicalDevice, C.VkPhysicalDevice](physicalDevice), (*C.uint32_t)(pQueueFamilyPropertyCount), (*C.VkQueueFamilyProperties2)(unsafe.Pointer(pQueueFamilyProperties)))
}

func (funcs *InstanceFuncs) GetPhysicalDeviceMemoryProperties2(physicalDevice PhysicalDevice, pMemoryProperties *PhysicalDeviceMemoryProperties2) {
	C.vkGetPhysicalDeviceMemoryProperties2(transmute[PhysicalDevice, C.VkPhysicalDevice](physicalDevice), (*C.VkPhysicalDeviceMemoryProperties2)(unsafe.Pointer(pMemoryProperties)))
}

func (funcs *InstanceFuncs) GetPhysicalDeviceProperties2(physicalDevice PhysicalDevice, properties *PhysicalDeviceProperties2) {
	C.vkGetPhysicalDeviceProperties2(transmute[PhysicalDevice, C.VkPhysicalDevice](physicalDevice), (*C.VkPhysicalDeviceProperties2)(unsafe.Pointer(properties)))
}

func (funcs *InstanceFuncs) GetPhysicalDeviceSurfaceSupportKHR(physicalDevice PhysicalDevice, queueFamilyIndex uint32, surface SurfaceKHR, pSupported *Bool32) error {
	return resultErr(Result(C.vkGetPhysicalDeviceSurfaceSupportKHR(transmute[PhysicalDevice, C.VkPhysicalDevice](physicalDevice), C.uint32_t(queueFamilyIndex), transmute[SurfaceKHR, C.VkSurfaceKHR](surface), (*C.VkBool32)(pSupported))))
}

func (funcs *InstanceFuncs) GetPhysicalDeviceFormatProperties2(physicalDevice PhysicalDevice, format Format, pFormatProperties *FormatProperties2) {
	C.vkGetPhysicalDeviceFormatProperties2(transmute[PhysicalDevice, C.VkPhysicalDevice](physicalDevice), C.VkFormat(format), (*C.VkFormatProperties2)(unsafe.Pointer(pFormatProperties)))
}

// TODO: C_ prefix is actually *really* annoying. We should pick something else. proc?
type DeviceFuncs struct {
	C_AcquireNextImage2KHR                     *[0]byte
	C_AllocateCommandBuffers                   *[0]byte
	C_AllocateDescriptorSets                   *[0]byte
	C_AllocateMemory                           *[0]byte
	C_BeginCommandBuffer                       *[0]byte
	C_BindBufferMemory2                        *[0]byte
	C_BindImageMemory2                         *[0]byte
	C_CmdBeginRendering                        *[0]byte
	C_CmdBindDescriptorSets                    *[0]byte
	C_CmdBindIndexBuffer                       *[0]byte
	C_CmdBindPipeline                          *[0]byte
	C_CmdBindShadersEXT                        *[0]byte
	C_CmdBuildAccelerationStructuresKHR        *[0]byte
	C_CmdClearAttachments                      *[0]byte
	C_CmdCopyBuffer2                           *[0]byte
	C_CmdCopyBufferToImage2                    *[0]byte
	C_CmdCopyImage2                            *[0]byte
	C_CmdCopyImageToBuffer2                    *[0]byte
	C_CmdDispatch                              *[0]byte
	C_CmdDraw                                  *[0]byte
	C_CmdDrawIndexed                           *[0]byte
	C_CmdEndRendering                          *[0]byte
	C_CmdExecuteCommands                       *[0]byte
	C_CmdPipelineBarrier2                      *[0]byte
	C_CmdPushConstants                         *[0]byte
	C_CmdSetAlphaToCoverageEnableEXT           *[0]byte
	C_CmdSetAlphaToOneEnableEXT                *[0]byte
	C_CmdSetColorBlendEnableEXT                *[0]byte
	C_CmdSetColorBlendEquationEXT              *[0]byte
	C_CmdSetColorWriteMaskEXT                  *[0]byte
	C_CmdSetCullModeEXT                        *[0]byte
	C_CmdSetDepthBiasEnableEXT                 *[0]byte
	C_CmdSetDepthBoundsTestEnableEXT           *[0]byte
	C_CmdSetDepthClampEnableEXT                *[0]byte
	C_CmdSetDepthCompareOpEXT                  *[0]byte
	C_CmdSetDepthTestEnableEXT                 *[0]byte
	C_CmdSetDepthWriteEnableEXT                *[0]byte
	C_CmdSetFrontFaceEXT                       *[0]byte
	C_CmdSetLogicOpEnableEXT                   *[0]byte
	C_CmdSetPolygonModeEXT                     *[0]byte
	C_CmdSetPrimitiveRestartEnableEXT          *[0]byte
	C_CmdSetPrimitiveTopologyEXT               *[0]byte
	C_CmdSetRasterizationSamplesEXT            *[0]byte
	C_CmdSetRasterizerDiscardEnableEXT         *[0]byte
	C_CmdSetRayTracingPipelineStackSizeKHR     *[0]byte
	C_CmdSetSampleMaskEXT                      *[0]byte
	C_CmdSetScissorWithCountEXT                *[0]byte
	C_CmdSetStencilTestEnableEXT               *[0]byte
	C_CmdSetVertexInputEXT                     *[0]byte
	C_CmdSetViewportWithCountEXT               *[0]byte
	C_CmdTraceRaysKHR                          *[0]byte
	C_CreateAccelerationStructureKHR           *[0]byte
	C_CreateBuffer                             *[0]byte
	C_CreateCommandPool                        *[0]byte
	C_CreateDescriptorPool                     *[0]byte
	C_CreateDescriptorSetLayout                *[0]byte
	C_CreateFence                              *[0]byte
	C_CreateImage                              *[0]byte
	C_CreateImageView                          *[0]byte
	C_CreatePipelineLayout                     *[0]byte
	C_CreateRayTracingPipelinesKHR             *[0]byte
	C_CreateSampler                            *[0]byte
	C_CreateSemaphore                          *[0]byte
	C_CreateShadersEXT                         *[0]byte
	C_CreateSwapchainKHR                       *[0]byte
	C_DestroyAccelerationStructureKHR          *[0]byte
	C_DestroyBuffer                            *[0]byte
	C_DestroyImage                             *[0]byte
	C_DestroyImageView                         *[0]byte
	C_DestroySampler                           *[0]byte
	C_DeviceWaitIdle                           *[0]byte
	C_EndCommandBuffer                         *[0]byte
	C_FreeMemory                               *[0]byte
	C_GetAccelerationStructureBuildSizesKHR    *[0]byte
	C_GetAccelerationStructureDeviceAddressKHR *[0]byte
	C_GetBufferDeviceAddress                   *[0]byte
	C_GetDeviceBufferMemoryRequirements        *[0]byte
	C_GetDeviceImageMemoryRequirements         *[0]byte
	C_GetDeviceQueue2                          *[0]byte
	C_GetMemoryHostPointerPropertiesEXT        *[0]byte
	C_GetRayTracingShaderGroupHandlesKHR       *[0]byte
	C_GetRayTracingShaderGroupStackSizeKHR     *[0]byte
	C_GetSwapchainImagesKHR                    *[0]byte
	C_MapMemory                                *[0]byte
	C_QueuePresentKHR                          *[0]byte
	C_QueueSubmit2                             *[0]byte
	C_QueueWaitIdle                            *[0]byte
	C_ResetFences                              *[0]byte
	C_UpdateDescriptorSets                     *[0]byte
	C_WaitForFences                            *[0]byte
	C_WaitSemaphores                           *[0]byte
}

func getDeviceProcAddr(device Device, name string) *[0]byte {
	return C.vkGetDeviceProcAddr(transmute[Device, C.VkDevice](device), (*C.char)(unsafe.Pointer(unsafe.StringData(name+"\x00"))))
}

// TODO: pass vkGetDeviceProcAddr
func (funcs *DeviceFuncs) Init(device Device) {
	rprocs := reflect.ValueOf(funcs).Elem()
	for i := range rprocs.NumField() {
		*rprocs.Field(i).Addr().Interface().(**[0]byte) = getDeviceProcAddr(device, "vk"+rprocs.Type().Field(i).Name[2:])
	}
}

func (funcs *DeviceFuncs) AcquireNextImage2KHR(device Device, acquireInfo *AcquireNextImageInfoKHR, imageIndex *uint32) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr(funcs.C_AcquireNextImage2KHR, C.uintptr_t(device), unsafe.Pointer(acquireInfo), unsafe.Pointer(imageIndex))))
}

func (funcs *DeviceFuncs) AllocateMemory(device Device, allocateInfo *MemoryAllocateInfo, allocator *AllocationCallbacks, memory *DeviceMemory) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_AllocateMemory, C.uintptr_t(device), unsafe.Pointer(allocateInfo), unsafe.Pointer(allocator), unsafe.Pointer(memory))))
}

func (funcs *DeviceFuncs) AllocateCommandBuffers(device Device, allocateInfo *CommandBufferAllocateInfo, commandBuffers *CommandBuffer) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr(funcs.C_AllocateCommandBuffers, C.uintptr_t(device), unsafe.Pointer(allocateInfo), unsafe.Pointer(commandBuffers))))
}

func (funcs *DeviceFuncs) AllocateDescriptorSets(device Device, allocateInfo *DescriptorSetAllocateInfo, descriptorSets *DescriptorSet) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr(funcs.C_AllocateDescriptorSets, C.uintptr_t(device), unsafe.Pointer(allocateInfo), unsafe.Pointer(descriptorSets))))
}

func (funcs *DeviceFuncs) BeginCommandBuffer(commandBuffer CommandBuffer, beginInfo *CommandBufferBeginInfo) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr(funcs.C_BeginCommandBuffer, C.uintptr_t(commandBuffer), unsafe.Pointer(beginInfo))))
}

func (funcs *DeviceFuncs) BindBufferMemory2(device Device, bindInfoCount uint32, bindInfos *BindBufferMemoryInfo) error {
	return resultErr(Result(C.vkcall_32_uintptr_32_ptr(funcs.C_BindBufferMemory2, C.uintptr_t(device), C.uint32_t(bindInfoCount), unsafe.Pointer(bindInfos))))
}

func (funcs *DeviceFuncs) BindImageMemory2(device Device, bindInfoCount uint32, bindInfos *BindImageMemoryInfo) error {
	return resultErr(Result(C.vkcall_32_uintptr_32_ptr(funcs.C_BindImageMemory2, C.uintptr_t(device), C.uint32_t(bindInfoCount), unsafe.Pointer(bindInfos))))
}

func (funcs *DeviceFuncs) CmdBeginRendering(commandBuffer CommandBuffer, renderingInfo *RenderingInfo) {
	C.vkcall_void_uintptr_ptr(funcs.C_CmdBeginRendering, C.uintptr_t(commandBuffer), unsafe.Pointer(renderingInfo))
}

func (funcs *DeviceFuncs) CmdBindDescriptorSets(commandBuffer CommandBuffer, pipelineBindPoint PipelineBindPoint, layout PipelineLayout, firstSet uint32, descriptorSetCount uint32, pDescriptorSets *DescriptorSet, dynamicOffsetCount uint32, pDynamicOffsets *uint32) {
	C.vkcall_void_uintptr_32_64_32_32_ptr_32_ptr(funcs.C_CmdBindDescriptorSets, C.uintptr_t(commandBuffer), C.uint32_t(pipelineBindPoint), C.uint64_t(layout), C.uint32_t(firstSet), C.uint32_t(descriptorSetCount), unsafe.Pointer(pDescriptorSets), C.uint32_t(dynamicOffsetCount), unsafe.Pointer(pDynamicOffsets))
}

func (funcs *DeviceFuncs) CmdBindIndexBuffer(commandBuffer CommandBuffer, buffer Buffer, offset DeviceSize, indexType IndexType) {
	C.vkcall_void_uintptr_64_64_32(funcs.C_CmdBindIndexBuffer, C.uintptr_t(commandBuffer), C.uint64_t(buffer), C.uint64_t(offset), C.uint32_t(indexType))
}

func (funcs *DeviceFuncs) CmdBindPipeline(commandBuffer CommandBuffer, pipelineBindPoint PipelineBindPoint, pipeline Pipeline) {
	C.vkcall_void_uintptr_32_64(funcs.C_CmdBindPipeline, C.uintptr_t(commandBuffer), C.uint32_t(pipelineBindPoint), C.uint64_t(pipeline))
}

func (funcs *DeviceFuncs) CmdBindShadersEXT(commandBuffer CommandBuffer, stageCount uint32, pStages *ShaderStageFlagBits, pShaders *ShaderEXT) {
	C.vkcall_void_uintptr_32_ptr_ptr(funcs.C_CmdBindShadersEXT, C.uintptr_t(commandBuffer), C.uint32_t(stageCount), unsafe.Pointer(pStages), unsafe.Pointer(pShaders))
}

func (funcs *DeviceFuncs) CmdBuildAccelerationStructuresKHR(commandBuffer CommandBuffer, infoCount uint32, pInfos *AccelerationStructureBuildGeometryInfoKHR, ppBuildRangeInfos **AccelerationStructureBuildRangeInfoKHR) {
	C.vkcall_void_uintptr_32_ptr_ptr(funcs.C_CmdBuildAccelerationStructuresKHR, C.uintptr_t(commandBuffer), C.uint32_t(infoCount), unsafe.Pointer(pInfos), unsafe.Pointer(ppBuildRangeInfos))
}

func (funcs *DeviceFuncs) CmdClearAttachments(commandBuffer CommandBuffer, attachmentCount uint32, attachments *ClearAttachment, rectCount uint32, rects *ClearRect) {
	C.vkcall_void_uintptr_32_ptr_32_ptr_noescape_nocallback(funcs.C_CmdClearAttachments, C.uintptr_t(commandBuffer), C.uint32_t(attachmentCount), unsafe.Pointer(attachments), C.uint32_t(rectCount), unsafe.Pointer(rects))
}

func (funcs *DeviceFuncs) CmdCopyBuffer2(commandBuffer CommandBuffer, pCopyBufferInfo *CopyBufferInfo2) {
	C.vkcall_void_uintptr_ptr(funcs.C_CmdCopyBuffer2, C.uintptr_t(commandBuffer), unsafe.Pointer(pCopyBufferInfo))
}

func (funcs *DeviceFuncs) CmdCopyBufferToImage2(commandBuffer CommandBuffer, pCopyBufferToImageInfo *CopyBufferToImageInfo2) {
	C.vkcall_void_uintptr_ptr(funcs.C_CmdCopyBufferToImage2, C.uintptr_t(commandBuffer), unsafe.Pointer(pCopyBufferToImageInfo))
}

func (funcs *DeviceFuncs) CmdCopyImage2(commandBuffer CommandBuffer, pCopyImageInfo *CopyImageInfo2) {
	C.vkcall_void_uintptr_ptr(funcs.C_CmdCopyImage2, C.uintptr_t(commandBuffer), unsafe.Pointer(pCopyImageInfo))
}

func (funcs *DeviceFuncs) CmdCopyImageToBuffer2(commandBuffer CommandBuffer, pCopyImageToBufferInfo *CopyImageToBufferInfo2) {
	C.vkcall_void_uintptr_ptr(funcs.C_CmdCopyImageToBuffer2, C.uintptr_t(commandBuffer), unsafe.Pointer(pCopyImageToBufferInfo))
}

func (funcs *DeviceFuncs) CmdDispatch(commandBuffer CommandBuffer, groupCountX, groupCountY, groupCountZ uint32) {
	C.vkcall_void_uintptr_32_32_32(funcs.C_CmdDispatch, C.uintptr_t(commandBuffer), C.uint32_t(groupCountX), C.uint32_t(groupCountY), C.uint32_t(groupCountZ))
}

func (funcs *DeviceFuncs) CmdDraw(commandBuffer CommandBuffer, vertexCount uint32, instanceCount uint32, firstVertex uint32, firstInstance uint32) {
	C.vkcall_void_uintptr_32_32_32_32(funcs.C_CmdDraw, C.uintptr_t(commandBuffer), C.uint32_t(vertexCount), C.uint32_t(instanceCount), C.uint32_t(firstVertex), C.uint32_t(firstInstance))
}

func (funcs *DeviceFuncs) CmdDrawIndexed(commandBuffer CommandBuffer, indexCount uint32, instanceCount uint32, firstIndex uint32, vertexOffset int32, firstInstance uint32) {
	C.vkcall_void_uintptr_32_32_32_32_32(funcs.C_CmdDrawIndexed, C.uintptr_t(commandBuffer), C.uint32_t(indexCount), C.uint32_t(instanceCount), C.uint32_t(firstIndex), C.uint32_t(vertexOffset), C.uint32_t(firstInstance))
}

func (funcs *DeviceFuncs) CmdEndRendering(commandBuffer CommandBuffer) {
	C.vkcall_void_uintptr(funcs.C_CmdEndRendering, C.uintptr_t(commandBuffer))
}

func (funcs *DeviceFuncs) CmdExecuteCommands(commandBuffer CommandBuffer, commandBufferCount uint32, pCommandBuffers *CommandBuffer) {
	C.vkcall_void_uintptr_32_ptr(funcs.C_CmdExecuteCommands, C.uintptr_t(commandBuffer), C.uint32_t(commandBufferCount), unsafe.Pointer(pCommandBuffers))
}

func (funcs *DeviceFuncs) CmdPipelineBarrier2(commandBuffer CommandBuffer, dependencyInfo *DependencyInfo) {
	C.vkcall_void_uintptr_ptr(funcs.C_CmdPipelineBarrier2, C.uintptr_t(commandBuffer), unsafe.Pointer(dependencyInfo))
}

func (funcs *DeviceFuncs) CmdPushConstants(commandBuffer CommandBuffer, layout PipelineLayout, stageFlags ShaderStageFlags, offset uint32, size uint32, pValues unsafe.Pointer) {
	C.vkcall_void_uintptr_64_32_32_32_ptr(funcs.C_CmdPushConstants, C.uintptr_t(commandBuffer), C.uint64_t(layout), C.uint32_t(stageFlags), C.uint32_t(offset), C.uint32_t(size), unsafe.Pointer(pValues))
}

func (funcs *DeviceFuncs) CmdSetAlphaToCoverageEnableEXT(commandBuffer CommandBuffer, alphaToCoverageEnable Bool32) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetAlphaToCoverageEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(alphaToCoverageEnable))
}

func (funcs *DeviceFuncs) CmdSetAlphaToOneEnableEXT(commandBuffer CommandBuffer, alphaToOneEnable Bool32) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetAlphaToOneEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(alphaToOneEnable))
}

func (funcs *DeviceFuncs) CmdSetColorBlendEnableEXT(commandBuffer CommandBuffer, firstAttachment uint32, attachmentCount uint32, pColorBlendEnables *Bool32) {
	C.vkcall_void_uintptr_32_32_ptr(funcs.C_CmdSetColorBlendEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(firstAttachment), C.uint32_t(attachmentCount), unsafe.Pointer(pColorBlendEnables))
}

func (funcs *DeviceFuncs) CmdSetColorBlendEquationEXT(commandBuffer CommandBuffer, firstAttachment uint32, attachmentCount uint32, pColorBlendEquations *ColorBlendEquationEXT) {
	C.vkcall_void_uintptr_32_32_ptr(funcs.C_CmdSetColorBlendEquationEXT, C.uintptr_t(commandBuffer), C.uint32_t(firstAttachment), C.uint32_t(attachmentCount), unsafe.Pointer(pColorBlendEquations))
}

func (funcs *DeviceFuncs) CmdSetColorWriteMaskEXT(commandBuffer CommandBuffer, firstAttachment uint32, attachmentCount uint32, pColorWriteMasks *ColorComponentFlags) {
	C.vkcall_void_uintptr_32_32_ptr(funcs.C_CmdSetColorWriteMaskEXT, C.uintptr_t(commandBuffer), C.uint32_t(firstAttachment), C.uint32_t(attachmentCount), unsafe.Pointer(pColorWriteMasks))
}

func (funcs *DeviceFuncs) CmdSetCullModeEXT(commandBuffer CommandBuffer, cullMode CullModeFlags) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetCullModeEXT, C.uintptr_t(commandBuffer), C.uint32_t(cullMode))
}

func (funcs *DeviceFuncs) CmdSetDepthBiasEnableEXT(commandBuffer CommandBuffer, depthBiasEnable Bool32) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetDepthBiasEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(depthBiasEnable))
}

func (funcs *DeviceFuncs) CmdSetDepthBoundsTestEnableEXT(commandBuffer CommandBuffer, depthBoundsTestEnable Bool32) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetDepthBoundsTestEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(depthBoundsTestEnable))
}

func (funcs *DeviceFuncs) CmdSetDepthClampEnableEXT(commandBuffer CommandBuffer, depthClampEnable Bool32) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetDepthClampEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(depthClampEnable))
}

func (funcs *DeviceFuncs) CmdSetDepthCompareOpEXT(commandBuffer CommandBuffer, depthCompareOp CompareOp) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetDepthCompareOpEXT, C.uintptr_t(commandBuffer), C.uint32_t(depthCompareOp))
}

func (funcs *DeviceFuncs) CmdSetDepthTestEnableEXT(commandBuffer CommandBuffer, depthTestEnable Bool32) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetDepthTestEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(depthTestEnable))
}

func (funcs *DeviceFuncs) CmdSetDepthWriteEnableEXT(commandBuffer CommandBuffer, depthWriteEnable Bool32) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetDepthWriteEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(depthWriteEnable))
}

func (funcs *DeviceFuncs) CmdSetFrontFaceEXT(commandBuffer CommandBuffer, frontFace FrontFace) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetFrontFaceEXT, C.uintptr_t(commandBuffer), C.uint32_t(frontFace))
}

func (funcs *DeviceFuncs) CmdSetLogicOpEnableEXT(commandBuffer CommandBuffer, logicOpEnable Bool32) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetLogicOpEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(logicOpEnable))
}

func (funcs *DeviceFuncs) CmdSetPolygonModeEXT(commandBuffer CommandBuffer, polygonMode PolygonMode) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetPolygonModeEXT, C.uintptr_t(commandBuffer), C.uint32_t(polygonMode))
}

func (funcs *DeviceFuncs) CmdSetPrimitiveRestartEnableEXT(commandBuffer CommandBuffer, primitiveRestartEnable Bool32) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetPrimitiveRestartEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(primitiveRestartEnable))
}

func (funcs *DeviceFuncs) CmdSetPrimitiveTopologyEXT(commandBuffer CommandBuffer, primitiveTopology PrimitiveTopology) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetPrimitiveTopologyEXT, C.uintptr_t(commandBuffer), C.uint32_t(primitiveTopology))
}

func (funcs *DeviceFuncs) CmdSetRasterizationSamplesEXT(commandBuffer CommandBuffer, rasterizationSamples SampleCountFlagBits) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetRasterizationSamplesEXT, C.uintptr_t(commandBuffer), C.uint32_t(rasterizationSamples))
}

func (funcs *DeviceFuncs) CmdSetRasterizerDiscardEnableEXT(commandBuffer CommandBuffer, rasterizerDiscardEnable Bool32) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetRasterizerDiscardEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(rasterizerDiscardEnable))
}

func (funcs *DeviceFuncs) CmdSetSampleMaskEXT(commandBuffer CommandBuffer, samples SampleCountFlagBits, pSampleMask *SampleMask) {
	C.vkcall_void_uintptr_32_ptr(funcs.C_CmdSetSampleMaskEXT, C.uintptr_t(commandBuffer), C.uint32_t(samples), unsafe.Pointer(pSampleMask))
}

func (funcs *DeviceFuncs) CmdSetScissorWithCountEXT(commandBuffer CommandBuffer, scissorCount uint32, pScissors *Rect2D) {
	C.vkcall_void_uintptr_32_ptr(funcs.C_CmdSetScissorWithCountEXT, C.uintptr_t(commandBuffer), C.uint32_t(scissorCount), unsafe.Pointer(pScissors))
}

func (funcs *DeviceFuncs) CmdSetStencilTestEnableEXT(commandBuffer CommandBuffer, stencilTestEnable Bool32) {
	C.vkcall_void_uintptr_32(funcs.C_CmdSetStencilTestEnableEXT, C.uintptr_t(commandBuffer), C.uint32_t(stencilTestEnable))
}

func (funcs *DeviceFuncs) CmdSetVertexInputEXT(commandBuffer CommandBuffer, vertexBindingDescriptionCount uint32, pVertexBindingDescriptions *VertexInputBindingDescription2EXT, vertexAttributeDescriptionCount uint32, pVertexAttributeDescriptions *VertexInputAttributeDescription2EXT) {
	C.vkcall_void_uintptr_32_ptr_32_ptr_noescape_nocallback(funcs.C_CmdSetVertexInputEXT, C.uintptr_t(commandBuffer), C.uint32_t(vertexBindingDescriptionCount), unsafe.Pointer(pVertexBindingDescriptions), C.uint32_t(vertexAttributeDescriptionCount), unsafe.Pointer(pVertexAttributeDescriptions))
}

func (funcs *DeviceFuncs) CmdSetViewportWithCountEXT(commandBuffer CommandBuffer, viewportCount uint32, pViewports *Viewport) {
	C.vkcall_void_uintptr_32_ptr(funcs.C_CmdSetViewportWithCountEXT, C.uintptr_t(commandBuffer), C.uint32_t(viewportCount), unsafe.Pointer(pViewports))
}

func (funcs *DeviceFuncs) CmdTraceRaysKHR(commandBuffer CommandBuffer, pRaygenShaderBindingTable *StridedDeviceAddressRegionKHR, pMissShaderBindingTable *StridedDeviceAddressRegionKHR, pHitShaderBindingTable *StridedDeviceAddressRegionKHR, pCallableShaderBindingTable *StridedDeviceAddressRegionKHR, width uint32, height uint32, depth uint32) {
	C.vkcall_void_uintptr_ptr_ptr_ptr_ptr_32_32_32_noescape_nocallback(funcs.C_CmdTraceRaysKHR, C.uintptr_t(commandBuffer), unsafe.Pointer(pRaygenShaderBindingTable), unsafe.Pointer(pMissShaderBindingTable), unsafe.Pointer(pHitShaderBindingTable), unsafe.Pointer(pCallableShaderBindingTable), C.uint32_t(width), C.uint32_t(height), C.uint32_t(depth))
}

func (funcs *DeviceFuncs) CreateAccelerationStructureKHR(device Device, pCreateInfo *AccelerationStructureCreateInfoKHR, pAllocator *AllocationCallbacks, pAccelerationStructure *AccelerationStructureKHR) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreateAccelerationStructureKHR, C.uintptr_t(device), unsafe.Pointer(pCreateInfo), unsafe.Pointer(pAllocator), unsafe.Pointer(pAccelerationStructure))))
}

func (funcs *DeviceFuncs) CreateBuffer(device Device, createInfo *BufferCreateInfo, allocator *AllocationCallbacks, buffer *Buffer) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreateBuffer, C.uintptr_t(device), unsafe.Pointer(createInfo), unsafe.Pointer(allocator), unsafe.Pointer(buffer))))
}

func (funcs *DeviceFuncs) CreateCommandPool(device Device, createInfo *CommandPoolCreateInfo, allocator *AllocationCallbacks, commandPool *CommandPool) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreateCommandPool, C.uintptr_t(device), unsafe.Pointer(createInfo), unsafe.Pointer(allocator), unsafe.Pointer(commandPool))))
}

func (funcs *DeviceFuncs) CreateDescriptorPool(device Device, createInfo *DescriptorPoolCreateInfo, allocator *AllocationCallbacks, descriptorPool *DescriptorPool) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreateDescriptorPool, C.uintptr_t(device), unsafe.Pointer(createInfo), unsafe.Pointer(allocator), unsafe.Pointer(descriptorPool))))
}

func (funcs *DeviceFuncs) CreateDescriptorSetLayout(device Device, createInfo *DescriptorSetLayoutCreateInfo, allocator *AllocationCallbacks, setLayout *DescriptorSetLayout) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreateDescriptorSetLayout, C.uintptr_t(device), unsafe.Pointer(createInfo), unsafe.Pointer(allocator), unsafe.Pointer(setLayout))))
}

func (funcs *DeviceFuncs) CreateFence(device Device, createInfo *FenceCreateInfo, allocator *AllocationCallbacks, pFence *Fence) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreateFence, C.uintptr_t(device), unsafe.Pointer(createInfo), unsafe.Pointer(allocator), unsafe.Pointer(pFence))))
}

func (funcs *DeviceFuncs) CreatePipelineLayout(device Device, createInfo *PipelineLayoutCreateInfo, allocator *AllocationCallbacks, pipelineLayout *PipelineLayout) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreatePipelineLayout, C.uintptr_t(device), unsafe.Pointer(createInfo), unsafe.Pointer(allocator), unsafe.Pointer(pipelineLayout))))
}

func (funcs *DeviceFuncs) CreateImage(device Device, createInfo *ImageCreateInfo, allocator *AllocationCallbacks, image *Image) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreateImage, C.uintptr_t(device), unsafe.Pointer(createInfo), unsafe.Pointer(allocator), unsafe.Pointer(image))))
}

func (funcs *DeviceFuncs) CreateImageView(device Device, createInfo *ImageViewCreateInfo, allocator *AllocationCallbacks, imageView *ImageView) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreateImageView, C.uintptr_t(device), unsafe.Pointer(createInfo), unsafe.Pointer(allocator), unsafe.Pointer(imageView))))
}

func (funcs *DeviceFuncs) CreateRayTracingPipelinesKHR(device Device, deferredOperation DeferredOperationKHR, pipelineCache PipelineCache, createInfoCount uint32, pCreateInfos *RayTracingPipelineCreateInfoKHR, pAllocator *AllocationCallbacks, pPipelines *Pipeline) error {
	return resultErr(Result(C.vkcall_32_uintptr_64_64_32_ptr_ptr_ptr(funcs.C_CreateRayTracingPipelinesKHR, C.uintptr_t(device), C.uint64_t(deferredOperation), C.uint64_t(pipelineCache), C.uint32_t(createInfoCount), unsafe.Pointer(pCreateInfos), unsafe.Pointer(pAllocator), unsafe.Pointer(pPipelines))))
}

func (funcs *DeviceFuncs) CreateSampler(device Device, pCreateInfo *SamplerCreateInfo, pAllocator *AllocationCallbacks, pSampler *Sampler) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreateSampler, C.uintptr_t(device), unsafe.Pointer(pCreateInfo), unsafe.Pointer(pAllocator), unsafe.Pointer(pSampler))))
}

func (funcs *DeviceFuncs) CreateSemaphore(device Device, createInfo *SemaphoreCreateInfo, allocator *AllocationCallbacks, pSemaphore *Semaphore) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreateSemaphore, C.uintptr_t(device), unsafe.Pointer(createInfo), unsafe.Pointer(allocator), unsafe.Pointer(pSemaphore))))
}

func (funcs *DeviceFuncs) CreateShadersEXT(device Device, createInfoCount uint32, createInfos *ShaderCreateInfoEXT, allocator *AllocationCallbacks, shaders *ShaderEXT) error {
	return resultErr(Result(C.vkcall_32_uintptr_32_ptr_ptr_ptr(funcs.C_CreateShadersEXT, C.uintptr_t(device), C.uint32_t(createInfoCount), unsafe.Pointer(createInfos), unsafe.Pointer(allocator), unsafe.Pointer(shaders))))
}

func (funcs *DeviceFuncs) CreateSwapchainKHR(device Device, createInfo *SwapchainCreateInfoKHR, allocator *AllocationCallbacks, swapchain *SwapchainKHR) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_ptr_ptr_noescape_nocallback(funcs.C_CreateSwapchainKHR, C.uintptr_t(device), unsafe.Pointer(createInfo), unsafe.Pointer(allocator), unsafe.Pointer(swapchain))))
}

func (funcs *DeviceFuncs) DestroyAccelerationStructureKHR(device Device, accelerationStructure AccelerationStructureKHR, pAllocator *AllocationCallbacks) {
	C.vkcall_void_uintptr_64_ptr(funcs.C_DestroyAccelerationStructureKHR, C.uintptr_t(device), C.uint64_t(accelerationStructure), unsafe.Pointer(pAllocator))
}

func (funcs *DeviceFuncs) DestroyBuffer(device Device, buffer Buffer, allocator *AllocationCallbacks) {
	C.vkcall_void_uintptr_64_ptr(funcs.C_DestroyBuffer, C.uintptr_t(device), C.uint64_t(buffer), unsafe.Pointer(allocator))
}

func (funcs *DeviceFuncs) DestroyImage(device Device, image Image, allocator *AllocationCallbacks) {
	C.vkcall_void_uintptr_64_ptr(funcs.C_DestroyImage, C.uintptr_t(device), C.uint64_t(image), unsafe.Pointer(allocator))
}

func (funcs *DeviceFuncs) DestroyImageView(device Device, imageView ImageView, allocator *AllocationCallbacks) {
	C.vkcall_void_uintptr_64_ptr(funcs.C_DestroyImageView, C.uintptr_t(device), C.uint64_t(imageView), unsafe.Pointer(allocator))
}

func (funcs *DeviceFuncs) DestroySampler(device Device, sampler Sampler, pAllocator *AllocationCallbacks) {
	C.vkcall_void_uintptr_64_ptr(funcs.C_DestroySampler, C.uintptr_t(device), C.uint64_t(sampler), unsafe.Pointer(pAllocator))
}

func (funcs *DeviceFuncs) EndCommandBuffer(commandBuffer CommandBuffer) error {
	return resultErr(Result(C.vkcall_32_uintptr(funcs.C_EndCommandBuffer, C.uintptr_t(commandBuffer))))
}

func (funcs *DeviceFuncs) FreeMemory(device Device, memory DeviceMemory, allocator *AllocationCallbacks) {
	C.vkcall_void_uintptr_64_ptr(funcs.C_FreeMemory, C.uintptr_t(device), C.uint64_t(memory), unsafe.Pointer(allocator))
}

func (funcs *DeviceFuncs) GetAccelerationStructureBuildSizesKHR(device Device, buildType AccelerationStructureBuildTypeKHR, pBuildInfo *AccelerationStructureBuildGeometryInfoKHR, pMaxPrimitiveCounts *uint32, pSizeInfo *AccelerationStructureBuildSizesInfoKHR) {
	C.vkcall_void_uintptr_32_ptr_ptr_ptr(funcs.C_GetAccelerationStructureBuildSizesKHR, C.uintptr_t(device), C.uint32_t(buildType), unsafe.Pointer(pBuildInfo), unsafe.Pointer(pMaxPrimitiveCounts), unsafe.Pointer(pSizeInfo))
}

func (funcs *DeviceFuncs) GetAccelerationStructureDeviceAddressKHR(device Device, pInfo *AccelerationStructureDeviceAddressInfoKHR) DeviceAddress {
	return DeviceAddress(C.vkcall_64_uintptr_ptr(funcs.C_GetAccelerationStructureDeviceAddressKHR, C.uintptr_t(device), unsafe.Pointer(pInfo)))
}

func (funcs *DeviceFuncs) GetBufferDeviceAddress(device Device, bufferDeviceAddressInfo *BufferDeviceAddressInfo) DeviceAddress {
	return DeviceAddress(C.vkcall_64_uintptr_ptr(funcs.C_GetBufferDeviceAddress, C.uintptr_t(device), unsafe.Pointer(bufferDeviceAddressInfo)))
}

func (funcs *DeviceFuncs) GetDeviceBufferMemoryRequirements(device Device, info *DeviceBufferMemoryRequirements, memoryRequirements *MemoryRequirements2) {
	C.vkcall_void_uintptr_ptr_ptr(funcs.C_GetDeviceBufferMemoryRequirements, C.uintptr_t(device), unsafe.Pointer(info), unsafe.Pointer(memoryRequirements))
}

func (funcs *DeviceFuncs) GetDeviceImageMemoryRequirements(device Device, info *DeviceImageMemoryRequirements, memoryRequirements *MemoryRequirements2) {
	C.vkcall_void_uintptr_ptr_ptr(funcs.C_GetDeviceImageMemoryRequirements, C.uintptr_t(device), unsafe.Pointer(info), unsafe.Pointer(memoryRequirements))
}

func (funcs *DeviceFuncs) GetDeviceQueue2(device Device, queueInfo *DeviceQueueInfo2, queue *Queue) {
	C.vkcall_void_uintptr_ptr_ptr(funcs.C_GetDeviceQueue2, C.uintptr_t(device), unsafe.Pointer(queueInfo), unsafe.Pointer(queue))
}

func (funcs *DeviceFuncs) GetMemoryHostPointerPropertiesEXT(device Device, handleType ExternalMemoryHandleTypeFlagBits, pHostPointer unsafe.Pointer, pMemoryHostPointerProperties *MemoryHostPointerPropertiesEXT) error {
	return resultErr(Result(C.vkcall_32_uintptr_32_ptr_ptr(funcs.C_GetMemoryHostPointerPropertiesEXT, C.uintptr_t(device), C.uint32_t(handleType), pHostPointer, unsafe.Pointer(pMemoryHostPointerProperties))))
}

func (funcs *DeviceFuncs) GetRayTracingShaderGroupHandlesKHR(device Device, pipeline Pipeline, firstGroup, groupCount uint32, dataSize int, pData unsafe.Pointer) error {
	return resultErr(Result(C.vkcall_32_uintptr_64_32_32_uintptr_ptr(funcs.C_GetRayTracingShaderGroupHandlesKHR, C.uintptr_t(device), C.uint64_t(pipeline), C.uint32_t(firstGroup), C.uint32_t(groupCount), C.uintptr_t(dataSize), pData)))
}

func (funcs *DeviceFuncs) GetRayTracingShaderGroupStackSizeKHR(device Device, pipeline Pipeline, group uint32, groupShader ShaderGroupShaderKHR) DeviceSize {
	return DeviceSize(C.vkcall_64_uintptr_64_32_32(funcs.C_GetRayTracingShaderGroupStackSizeKHR, C.uintptr_t(device), C.uint64_t(pipeline), C.uint32_t(group), C.uint32_t(groupShader)))
}

func (funcs *DeviceFuncs) GetSwapchainImagesKHR(device Device, swapchain SwapchainKHR, swapchainImageCount *uint32, swapchainImages *Image) error {
	return resultErr(Result(C.vkcall_32_uintptr_64_ptr_ptr(funcs.C_GetSwapchainImagesKHR, C.uintptr_t(device), C.uint64_t(swapchain), unsafe.Pointer(swapchainImageCount), unsafe.Pointer(swapchainImages))))
}

func (funcs *DeviceFuncs) MapMemory(device Device, memory DeviceMemory, offset DeviceSize, size DeviceSize, flags MemoryMapFlags, data *unsafe.Pointer) error {
	return resultErr(Result(C.vkMapMemory(transmute[Device, C.VkDevice](device), transmute[DeviceMemory, C.VkDeviceMemory](memory), C.VkDeviceSize(offset), C.VkDeviceSize(size), C.VkMemoryMapFlags(flags), data)))
}

func (funcs *DeviceFuncs) QueuePresentKHR(queue Queue, presentInfo *PresentInfoKHR) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr(funcs.C_QueuePresentKHR, C.uintptr_t(queue), unsafe.Pointer(presentInfo))))
}

func (funcs *DeviceFuncs) QueueSubmit2(queue Queue, submitCount uint32, submitInfos *SubmitInfo2, fence Fence) error {
	return resultErr(Result(C.vkcall_32_uintptr_32_ptr_64(funcs.C_QueueSubmit2, C.uintptr_t(queue), C.uint32_t(submitCount), unsafe.Pointer(submitInfos), C.uint64_t(fence))))
}

func (funcs *DeviceFuncs) QueueWaitIdle(queue Queue) error {
	return resultErr(Result(C.vkcall_32_uintptr(funcs.C_QueueWaitIdle, C.uintptr_t(queue))))
}

func (funcs *DeviceFuncs) ResetFences(device Device, fenceCount uint32, pFences *Fence) error {
	return resultErr(Result(C.vkcall_32_uintptr_32_ptr(funcs.C_ResetFences, C.uintptr_t(device), C.uint32_t(fenceCount), unsafe.Pointer(pFences))))
}

func (funcs *DeviceFuncs) UpdateDescriptorSets(device Device, descriptorWriteCount uint32, pDescriptorWrites *WriteDescriptorSet, descriptorCopyCount uint32, pDescriptorCopies *CopyDescriptorSet) {
	C.vkcall_void_uintptr_32_ptr_32_ptr_noescape_nocallback(funcs.C_UpdateDescriptorSets, C.uintptr_t(device), C.uint32_t(descriptorWriteCount), unsafe.Pointer(pDescriptorWrites), C.uint32_t(descriptorCopyCount), unsafe.Pointer(&pDescriptorCopies))
}

func (funcs *DeviceFuncs) WaitForFences(device Device, fenceCount uint32, pFences *Fence, waitAll Bool32, timeout uint64) error {
	return resultErr(Result(C.vkcall_32_uintptr_32_ptr_32_64(funcs.C_WaitForFences, C.uintptr_t(device), C.uint32_t(fenceCount), unsafe.Pointer(pFences), C.uint32_t(waitAll), C.uint64_t(timeout))))
}

func (funcs *DeviceFuncs) WaitSemaphores(device Device, pWaitInfo *SemaphoreWaitInfo, timeout uint64) error {
	return resultErr(Result(C.vkcall_32_uintptr_ptr_64_noescape_nocallback(funcs.C_WaitSemaphores, C.uintptr_t(device), unsafe.Pointer(pWaitInfo), C.uint64_t(timeout))))
}
