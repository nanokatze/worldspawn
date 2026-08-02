// TODO: generate String() for the enums? *Flags are bitmasks

//go:generate go run mkfuncs.go -o funcs.go vk.xml
//go:generate go run mktypes.go -o types.go vk.xml

//go:generate stringer -type Format -trimprefix FORMAT_
//go:generate stringer -type Result

package vk

import (
	"unsafe"
)

const NULL_HANDLE = 0

type Bool32 uint32

const (
	FALSE Bool32 = 0
	TRUE  Bool32 = 1
)

type DeviceSize uint64

const API_VERSION_1_4 = 1<<22 | 4<<12

type SampleMask uint32

type PFN_vkAllocationFunction *[0]byte
type PFN_vkReallocationFunction *[0]byte
type PFN_vkFreeFunction *[0]byte
type PFN_vkInternalAllocationNotification *[0]byte
type PFN_vkInternalFreeNotification *[0]byte
type PFN_vkDebugReportCallbackEXT *[0]byte
type PFN_vkDebugUtilsMessengerCallbackEXT *[0]byte
type PFN_vkDeviceMemoryReportCallbackEXT *[0]byte
type PFN_vkGetInstanceProcAddrLUNARG *[0]byte

const WHOLE_SIZE = ^uint64(0)

const MAX_PHYSICAL_DEVICE_NAME_SIZE = 256

const MAX_EXTENSION_NAME_SIZE = 256

const MAX_DESCRIPTION_SIZE = 256

const MAX_MEMORY_TYPES = 32

const MAX_MEMORY_HEAPS = 16

const LOD_CLAMP_NONE = float32(1000)

const UUID_SIZE = 16

const LUID_SIZE = 8

const QUEUE_FAMILY_EXTERNAL = ^uint32(1)

const MAX_DEVICE_GROUP_SIZE = 32

const SHADER_UNUSED_KHR = ^uint32(0)

const MAX_GLOBAL_PRIORITY_SIZE = 16

const MAX_DRIVER_NAME_SIZE = 256

const MAX_DRIVER_INFO_SIZE = 256

const MAX_SHADER_MODULE_IDENTIFIER_SIZE_EXT = 32

const MAX_PIPELINE_BINARY_KEY_SIZE_KHR = 32

const MAX_PHYSICAL_DEVICE_DATA_GRAPH_OPERATION_SET_NAME_SIZE_ARM = 128

const DATA_GRAPH_MODEL_TOOLCHAIN_VERSION_LENGTH_QCOM = 3

const MAX_DATA_GRAPH_TOSA_NAME_SIZE_ARM = 128

const MAX_TENSOR_CREATE_INFO_ROLLING_BACKING_WRAP_COUNT_ARM = 4

type ClearColorValue [4]uint32

type ClearValue [4]uint32

type DeviceOrHostAddressConstKHR uint64

type DeviceOrHostAddressKHR uint64

type DeviceAddress uint64

type PerformanceValueDataINTEL uint64

type PipelineExecutableStatisticValueKHR uint64

type DescriptorDataEXT uint64

type ClusterAccelerationStructureOpInputNV uint64

type ResourceDescriptorDataEXT unsafe.Pointer

type DescriptorMappingSourceDataEXT struct{ _ uint64 } // broken, do not use

// the following are borked

type CudaModuleCreateInfoNV uint64
type CudaFunctionCreateInfoNV uint64
type CudaLaunchInfoNV uint64
type PhysicalDeviceCudaKernelLaunchFeaturesNV uint64
type PhysicalDeviceCudaKernelLaunchPropertiesNV uint64

const STRUCTURE_TYPE_CUDA_MODULE_CREATE_INFO_NV = 99999999
const STRUCTURE_TYPE_CUDA_FUNCTION_CREATE_INFO_NV = 99999999
const STRUCTURE_TYPE_CUDA_LAUNCH_INFO_NV = 99999999
const STRUCTURE_TYPE_PHYSICAL_DEVICE_CUDA_KERNEL_LAUNCH_FEATURES_NV = 99999999
const STRUCTURE_TYPE_PHYSICAL_DEVICE_CUDA_KERNEL_LAUNCH_PROPERTIES_NV = 99999999

const sizeofAccelerationStructureGeometryDataKHR = max(
	unsafe.Sizeof(AccelerationStructureGeometryTrianglesDataKHR{}),
	unsafe.Sizeof(AccelerationStructureGeometryAabbsDataKHR{}),
	unsafe.Sizeof(AccelerationStructureGeometryInstancesDataKHR{}))

type AccelerationStructureGeometryDataKHR [(sizeofAccelerationStructureGeometryDataKHR + 8 - 1) / 8]uint64

func AccelerationStructureGeometryDataTriangles(triangles AccelerationStructureGeometryTrianglesDataKHR) AccelerationStructureGeometryDataKHR {
	var dst AccelerationStructureGeometryDataKHR
	*(*AccelerationStructureGeometryTrianglesDataKHR)(unsafe.Pointer(&dst)) = triangles
	return dst
}

func AccelerationStructureGeometryDataInstances(instances AccelerationStructureGeometryInstancesDataKHR) AccelerationStructureGeometryDataKHR {
	var dst AccelerationStructureGeometryDataKHR
	*(*AccelerationStructureGeometryInstancesDataKHR)(unsafe.Pointer(&dst)) = instances
	return dst
}

// TODO: this is incorrect and we should figure the correct size out
type AccelerationStructureMotionInstanceDataNV [1]uint64

type IndirectExecutionSetInfoEXT unsafe.Pointer

type IndirectCommandsTokenDataEXT unsafe.Pointer

// TODO: rename tmp in the result to something else

// TODO: we could use a generic function for this

// TODO: prefix these helpers with Make?

// TODO: remove this
func transmute[T1, T2 any](v T1) T2 {
	var a [1]struct{}
	_ = a[unsafe.Sizeof(*new(T1))-unsafe.Sizeof(*new(T2))]

	return *(*T2)(unsafe.Pointer(&v))
}
