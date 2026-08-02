package gpu

import (
	"maps"
	"runtime"
	"slices"
	"sync"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: split this package up into the base gpu package, gpuruntime, maybe
// gpuimage, maybe more. gpuruntime would export things that external users
// might interact with, but usually won't need to (e.g. Job, JobInfo,
// CommandQueue) so it's for example possible to implement presentation
// externally.

// TODO: alias or replace definitions of vulkan types we use so that most
// users won't need to import worldspawn/gpu/vk

// TODO: add GPUGODEBUG env var and various flags

// TODO: let third party packages sneak in the instance exts and device features
// and extensions so we can move swapchain/presentation to its own package.

/*
type _Device struct {
	vk                vk.Device
	topology          queues
	haveExts map[string]struct{}
	haveFeatures   map[string]struct{}
}

func (device *_Device) Vk() vk.Device { return device.vk }

func (device *_Device) HaveExtension(ext string) bool { panic("not implemented") }

func (device *_Device) HaveFeature(feature string) bool { panic("not implemented") }
*/

var (
	appName    string = "Worldspawn" // TODO: should not have a default. These should be set by the linker.
	appVersion uint32
)

// TODO: rename
type extensions map[string]bool

// TODO: record where the ext was requested
func (exts extensions) Add(ext string, need bool) {
	if !exts[ext] {
		exts[ext] = need
	}
}

// TODO: plop these into a struct
var (
	wantInstanceExts   = extensions{}
	wantDeviceExts     = extensions{}
	wantDeviceFeatures = extensions{}
)

// TODO: prefix these with WantVulkan..., e.g. WantVulkanDeviceExtension?

func WantInstanceExtension(extension string) {
	wantInstanceExts.Add(extension, false)
}

func WantDeviceExtension(extension string) {
	wantDeviceExts.Add(extension, false)
}

func WantDeviceFeature(feature string) {
	wantDeviceFeatures.Add(feature, false)
}

var instance vk.Instance
var physicalDevice vk.PhysicalDevice
var topology *deviceTopology
var device vk.Device

var DescriptorSetLayout vk.DescriptorSetLayout
var descriptorSet vk.DescriptorSet
var pipelineLayout vk.PipelineLayout

var vkFns struct {
	vk.InstanceFuncs
	vk.DeviceFuncs
}

var gpuInitOnce sync.Once

// TODO: should be a method on _Device
func QueueFamilies(flags vk.QueueFlags) queueFamilyMask {
	return topology.QueueFamilies(flags)
}

// TODO: kill
func BestQueueFamily(flags vk.QueueFlags) int {
	return topology.MinimumCapable(flags)
}

// TODO: these should probably folded into Device() and Device() be renamed to Init()
func Instance() vk.Instance {
	gpuInit()
	return instance
}

func PhysicalDevice() vk.PhysicalDevice {
	gpuInit()
	return physicalDevice
}

// TODO: fix
func Device() vk.Device {
	gpuInit()
	return device
}

func gpuInit() {
	gpuInitOnce.Do(func() {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		WantInstanceExtension("VK_EXT_debug_utils")

		instanceExts := slices.Sorted(maps.Keys(wantInstanceExts))

		must(vk.CreateInstance(pinned(&pinner, &vk.InstanceCreateInfo{
			SType: vk.STRUCTURE_TYPE_INSTANCE_CREATE_INFO,
			PApplicationInfo: pinned(&pinner, &vk.ApplicationInfo{
				SType:              vk.STRUCTURE_TYPE_APPLICATION_INFO,
				PApplicationName:   pinnedCString(&pinner, appName),
				ApplicationVersion: appVersion,
				ApiVersion:         vk.API_VERSION_1_4,
			}),
			EnabledExtensionCount:   uint32(len(instanceExts)),
			PPEnabledExtensionNames: pinnedCStringSlice(&pinner, instanceExts),
		}), nil, &instance))

		vkFns.InstanceFuncs.Init(instance)

		physicalDevices, err := enumerate(func(len *uint32, data *vk.PhysicalDevice) error {
			return vkFns.EnumeratePhysicalDevices(instance, len, data)
		})
		must(err)
		physicalDevice = physicalDevices[0]

		props := vk.PhysicalDeviceProperties2{
			SType: vk.STRUCTURE_TYPE_PHYSICAL_DEVICE_PROPERTIES_2,
		}
		vkFns.GetPhysicalDeviceProperties2(physicalDevice, &props)

		println("GPU:", byteSliceToString(props.DeviceName[:]))

		// TODO: move default feature enables somewhere else.
		// TODO: request almost all core features probably. Or consult a profile

		WantDeviceFeature("ImageCubeArray")
		WantDeviceFeature("SamplerAnisotropy")
		WantDeviceFeature("ShaderInt64")
		WantDeviceFeature("ShaderInt16")

		WantDeviceFeature("StorageBuffer16BitAccess")
		WantDeviceFeature("StoragePushConstant16")

		WantDeviceFeature("StorageBuffer8BitAccess")
		WantDeviceFeature("StoragePushConstant8")
		WantDeviceFeature("ShaderFloat16")
		WantDeviceFeature("ShaderInt8")
		WantDeviceFeature("ShaderSampledImageArrayNonUniformIndexing")
		WantDeviceFeature("ShaderStorageImageArrayNonUniformIndexing")
		WantDeviceFeature("DescriptorBindingSampledImageUpdateAfterBind")
		WantDeviceFeature("DescriptorBindingStorageImageUpdateAfterBind")
		WantDeviceFeature("DescriptorBindingUpdateUnusedWhilePending")
		WantDeviceFeature("DescriptorBindingPartiallyBound")
		WantDeviceFeature("DescriptorBindingVariableDescriptorCount")
		WantDeviceFeature("RuntimeDescriptorArray")
		WantDeviceFeature("ScalarBlockLayout")
		WantDeviceFeature("TimelineSemaphore")
		WantDeviceFeature("BufferDeviceAddress")
		WantDeviceFeature("VulkanMemoryModel")
		WantDeviceFeature("VulkanMemoryModelDeviceScope")
		// Not sure we actually need these
		// DeviceFeature("VulkanMemoryModelAvailabilityVisibilityChains")

		// TODO: we also might need demote and discard, subgroup size control,
		// compute full subgroups
		WantDeviceFeature("Synchronization2")
		WantDeviceFeature("DynamicRendering")
		WantDeviceFeature("Maintenance4")

		WantDeviceFeature("Maintenance5")
		WantDeviceFeature("Maintenance6")

		WantDeviceExtension("VK_KHR_maintenance7")
		WantDeviceFeature("Maintenance7")

		WantDeviceExtension("VK_KHR_maintenance8")
		WantDeviceFeature("Maintenance8")

		// We depend maintenance9 to avoid specifying sharing mode for buffers and
		wantDeviceExts.Add("VK_KHR_maintenance9", true)
		wantDeviceFeatures.Add("Maintenance9", true)

		wantDeviceExts.Add("VK_KHR_device_address_commands", true)
		wantDeviceFeatures.Add("DeviceAddressCommands", true)

		wantDeviceExts.Add("VK_KHR_internally_synchronized_queues", true)
		wantDeviceFeatures.Add("InternallySynchronizedQueues", true)

		wantDeviceExts.Add("VK_EXT_shader_object", true)
		wantDeviceFeatures.Add("ShaderObject", true)

		WantDeviceExtension("VK_EXT_image_view_min_lod")
		WantDeviceFeature("MinLod")

		WantDeviceExtension("VK_EXT_mesh_shader")
		WantDeviceFeature("TaskShader")
		WantDeviceFeature("MeshShader")

		WantDeviceExtension("VK_KHR_deferred_host_operations")

		WantDeviceExtension("VK_KHR_acceleration_structure")
		WantDeviceFeature("AccelerationStructure")

		WantDeviceExtension("VK_KHR_pipeline_library")

		WantDeviceExtension("VK_EXT_pipeline_library_group_handles")
		WantDeviceFeature("PipelineLibraryGroupHandles")

		WantDeviceExtension("VK_KHR_ray_tracing_pipeline")
		WantDeviceFeature("RayTracingPipeline")

		WantDeviceExtension("VK_KHR_ray_query")
		WantDeviceFeature("RayQuery")

		WantDeviceExtension("VK_KHR_ray_tracing_maintenance1")
		WantDeviceFeature("RayTracingMaintenance1")

		WantDeviceExtension("VK_KHR_ray_tracing_position_fetch")
		WantDeviceFeature("RayTracingPositionFetch")

		// WantDeviceExtension("VK_EXT_external_memory_host")

		availableExtensionsSlice, err := enumerate(func(len *uint32, data *vk.ExtensionProperties) error {
			return vkFns.EnumerateDeviceExtensionProperties(physicalDevice, len, data)
		})
		availableExtensions := maps.Collect(func(yield func(string, struct{}) bool) {
			for _, ext := range availableExtensionsSlice {
				yield(byteSliceToString(ext.ExtensionName[:]), struct{}{})
			}
		})

		var enabledDeviceExtensionsSlice []string
		for _, ext := range slices.Sorted(maps.Keys(wantDeviceExts)) {
			if _, ok := availableExtensions[ext]; !ok {
				if wantDeviceExts[ext] {
					// TODO: eagerly output as many as we can and bail only
					// later.
					panic("don't have required extension " + ext)
				}
				continue
			}
			enabledDeviceExtensionsSlice = append(enabledDeviceExtensionsSlice, ext)
		}

		var availableFeatures deviceFeatures
		availableFeatures.init(false)
		pinner.Pin(&availableFeatures)
		vkFns.GetPhysicalDeviceFeatures2(physicalDevice, &availableFeatures.PhysicalDeviceFeatures2)

		var enabledFeatures deviceFeatures
		// TODO: rewrite this to be less ugly
		for _, feature := range slices.Sorted(maps.Keys(wantDeviceFeatures)) {
			if !availableFeatures.Get(feature) {
				if wantDeviceFeatures[feature] {
					panic("don't have required feature " + feature)
				}
				continue
			}
			enabledFeatures.Set(feature)
		}
		enabledFeatures.init(true)

		topology = defaultQueues()

		queueCreateInfos := slices.Collect(func(yield func(vk.DeviceQueueCreateInfo) bool) {
			for family, count := range topology.counts {
				if count == 0 {
					continue
				}
				prios := make([]float32, count)
				yield(vk.DeviceQueueCreateInfo{
					SType:            vk.STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO,
					Flags:            vk.DeviceQueueCreateFlags(vk.DEVICE_QUEUE_CREATE_INTERNALLY_SYNCHRONIZED_BIT_KHR),
					QueueFamilyIndex: uint32(family),
					QueueCount:       uint32(count),
					PQueuePriorities: pinnedSliceData(&pinner, prios),
				})
			}
		})

		pinner.Pin(&enabledFeatures)

		must(vkFns.CreateDevice(physicalDevice, &vk.DeviceCreateInfo{
			SType:                   vk.STRUCTURE_TYPE_DEVICE_CREATE_INFO,
			PNext:                   unsafe.Pointer(&enabledFeatures.PhysicalDeviceFeatures2),
			QueueCreateInfoCount:    uint32(len(queueCreateInfos)),
			PQueueCreateInfos:       pinnedSliceData(&pinner, queueCreateInfos),
			EnabledExtensionCount:   uint32(len(enabledDeviceExtensionsSlice)),
			PPEnabledExtensionNames: pinnedCStringSlice(&pinner, enabledDeviceExtensionsSlice),
		}, &device))

		vkFns.DeviceFuncs.Init(device)

		for family := range ones32(QueueFamilies(0)) {
			for i := range topology.counts[family] {
				newDeviceQueue(family, i)
			}
		}

		must(vkFns.CreateDescriptorSetLayout(device, &vk.DescriptorSetLayoutCreateInfo{
			SType: vk.STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_CREATE_INFO,
			PNext: unsafe.Pointer(pinned(&pinner, &vk.DescriptorSetLayoutBindingFlagsCreateInfo{
				SType:        vk.STRUCTURE_TYPE_DESCRIPTOR_SET_LAYOUT_BINDING_FLAGS_CREATE_INFO,
				BindingCount: 3,
				PBindingFlags: pinnedSliceData(&pinner, []vk.DescriptorBindingFlags{
					vk.DescriptorBindingFlags(vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_UNUSED_WHILE_PENDING_BIT | vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT),
					vk.DescriptorBindingFlags(vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_UNUSED_WHILE_PENDING_BIT | vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT),
					vk.DescriptorBindingFlags(vk.DESCRIPTOR_BINDING_UPDATE_AFTER_BIND_BIT | vk.DESCRIPTOR_BINDING_UPDATE_UNUSED_WHILE_PENDING_BIT | vk.DESCRIPTOR_BINDING_PARTIALLY_BOUND_BIT),
				}),
			})),
			Flags:        vk.DescriptorSetLayoutCreateFlags(vk.DESCRIPTOR_SET_LAYOUT_CREATE_UPDATE_AFTER_BIND_POOL_BIT),
			BindingCount: 3,
			PBindings: pinnedSliceData(&pinner, []vk.DescriptorSetLayoutBinding{
				{
					Binding:         0,
					DescriptorType:  vk.DESCRIPTOR_TYPE_SAMPLER,
					DescriptorCount: 2e3,
					StageFlags:      vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
				},
				{
					Binding:         1,
					DescriptorType:  vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE,
					DescriptorCount: 1e6,
					StageFlags:      vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
				},
				{
					Binding:         2,
					DescriptorType:  vk.DESCRIPTOR_TYPE_STORAGE_IMAGE,
					DescriptorCount: 1e6,
					StageFlags:      vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
				},
			}),
		}, nil, &DescriptorSetLayout))

		must(vkFns.CreatePipelineLayout(device, &vk.PipelineLayoutCreateInfo{
			SType:                  vk.STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
			SetLayoutCount:         1,
			PSetLayouts:            pinned(&pinner, &DescriptorSetLayout),
			PushConstantRangeCount: 1,
			PPushConstantRanges: pinned(&pinner, &vk.PushConstantRange{
				StageFlags: vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
				Offset:     0,
				Size:       maxShaderArgsSize,
			}),
		}, nil, &pipelineLayout))

		var descriptorPool vk.DescriptorPool
		must(vkFns.CreateDescriptorPool(device, &vk.DescriptorPoolCreateInfo{
			SType:         vk.STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
			Flags:         vk.DescriptorPoolCreateFlags(vk.DESCRIPTOR_POOL_CREATE_UPDATE_AFTER_BIND_BIT),
			MaxSets:       1,
			PoolSizeCount: 3,
			PPoolSizes: pinnedSliceData(&pinner, []vk.DescriptorPoolSize{
				{Type: vk.DESCRIPTOR_TYPE_SAMPLER, DescriptorCount: 2e3},
				{Type: vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE, DescriptorCount: 1e6},
				{Type: vk.DESCRIPTOR_TYPE_STORAGE_IMAGE, DescriptorCount: 1e6},
			}),
		}, nil, &descriptorPool))

		must(vkFns.AllocateDescriptorSets(device, &vk.DescriptorSetAllocateInfo{
			SType:              vk.STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
			DescriptorPool:     descriptorPool,
			DescriptorSetCount: 1,
			PSetLayouts:        pinned(&pinner, &DescriptorSetLayout),
		}, &descriptorSet))
	})
}

// TODO: rename to something better
func enumerate[T any](f func(len *uint32, data *T) error) ([]T, error) {
	return enumerate2(nil, f)
}

// TODO: rename to something better
func enumerate2[T any](init func([]T), f func(len *uint32, data *T) error) ([]T, error) {
	var len uint32
	if err := f(&len, nil); err != nil {
		return nil, err
	}
	data := make([]T, int(len))
	if init != nil {
		init(data)
	}
	if err := f(&len, unsafe.SliceData(data)); err != nil {
		return nil, err
	}
	return data[:len], nil
}
