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
	enabledExtensions map[string]struct{}
	enabledFeatures   map[string]struct{}
}

func (device *_Device) Vk() vk.Device { return device.vk }

func (device *_Device) HaveExtension(ext string) bool { panic("not implemented") }

func (device *_Device) HaveFeature(feature string) bool { panic("not implemented") }
*/

// TODO: record stack traces of where these features were enabled?
var requestedDeviceExtensions = map[string]struct{}{}

var requestedDeviceFeatures = map[string]bool{}

// TODO: make these device options? So we'd e.g. do SetDeviceOption()
func DeviceExtension(extension string, required bool) {
	requestedDeviceExtensions[extension] = struct{}{}
}

func DeviceFeature(feature string, required bool) {
	if !requestedDeviceFeatures[feature] {
		requestedDeviceFeatures[feature] = required
	}
}

// type requirementsMap map[string]bool

var instance vk.Instance
var physicalDevice vk.PhysicalDevice
var queueFamilies *queues
var device vk.Device

var descriptorSetLayout vk.DescriptorSetLayout
var descriptorSet vk.DescriptorSet
var pipelineLayout vk.PipelineLayout

var vkFns struct {
	vk.InstanceFuncs
	vk.DeviceFuncs
}

var gpuInitOnce sync.Once

func gpuInit() {
	gpuInitOnce.Do(func() {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		layers, err := enumerate(vk.EnumerateInstanceLayerProperties)
		must(err)

		var instanceLayers []string
		for _, layer := range layers {
			switch byteSliceToString(layer.LayerName[:]) {
			case "VK_LAYER_KHRONOS_shader_object":
				// log.Println("VK_LAYER_KHRONOS_shader_object found, enabling")
				// instanceLayers = append(instanceLayers, "VK_LAYER_KHRONOS_shader_object")
			}
		}

		// TODO: we want to make additional extensions require-able by other
		// components. Do we let external things register a callback and
		// populate extensions here?

		instanceExtensions := map[string]struct{}{
			"VK_KHR_surface":         {},
			"VK_KHR_wayland_surface": {},
			"VK_KHR_xlib_surface":    {},
		}
		if true {
			instanceExtensions["VK_EXT_debug_utils"] = struct{}{}
		}

		instanceExtensionsSlice := slices.Sorted(maps.Keys(instanceExtensions))

		must(vk.CreateInstance(pinned(&pinner, &vk.InstanceCreateInfo{
			SType: vk.STRUCTURE_TYPE_INSTANCE_CREATE_INFO,
			PApplicationInfo: pinned(&pinner, &vk.ApplicationInfo{
				SType:            vk.STRUCTURE_TYPE_APPLICATION_INFO,
				PApplicationName: pinnedCString(&pinner, "Worldspawn"),
				ApiVersion:       vk.API_VERSION_1_4,
			}),
			EnabledLayerCount:       uint32(len(instanceLayers)),
			PPEnabledLayerNames:     pinnedCStringSlice(&pinner, instanceLayers),
			EnabledExtensionCount:   uint32(len(instanceExtensionsSlice)),
			PPEnabledExtensionNames: pinnedCStringSlice(&pinner, instanceExtensionsSlice),
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

		DeviceFeature("ImageCubeArray", true)
		DeviceFeature("SamplerAnisotropy", true)
		DeviceFeature("ShaderInt64", true)
		DeviceFeature("ShaderInt16", true)

		DeviceFeature("StorageBuffer16BitAccess", true)
		DeviceFeature("StoragePushConstant16", true)

		DeviceFeature("StorageBuffer8BitAccess", true)
		DeviceFeature("StoragePushConstant8", true)
		DeviceFeature("ShaderFloat16", true)
		DeviceFeature("ShaderInt8", true)
		DeviceFeature("ShaderSampledImageArrayNonUniformIndexing", true)
		DeviceFeature("ShaderStorageImageArrayNonUniformIndexing", true)
		DeviceFeature("DescriptorBindingSampledImageUpdateAfterBind", true)
		DeviceFeature("DescriptorBindingStorageImageUpdateAfterBind", true)
		DeviceFeature("DescriptorBindingUpdateUnusedWhilePending", true)
		DeviceFeature("DescriptorBindingPartiallyBound", true)
		DeviceFeature("DescriptorBindingVariableDescriptorCount", true)
		DeviceFeature("RuntimeDescriptorArray", true)
		DeviceFeature("ScalarBlockLayout", true)
		DeviceFeature("TimelineSemaphore", true)
		DeviceFeature("BufferDeviceAddress", true)
		DeviceFeature("VulkanMemoryModel", true)
		// Not sure we actually need these
		// DeviceFeature("VulkanMemoryModelDeviceScope", true)
		// DeviceFeature("VulkanMemoryModelAvailabilityVisibilityChains", true)

		// TODO: we also might need demote and discard, subgroup size control,
		// compute full subgroups
		DeviceFeature("Synchronization2", true)
		DeviceFeature("DynamicRendering", true)
		DeviceFeature("Maintenance4", true)

		DeviceFeature("Maintenance5", true)
		DeviceFeature("Maintenance6", true)

		DeviceExtension("VK_KHR_maintenance7", true)
		DeviceFeature("Maintenance7", true)

		DeviceExtension("VK_KHR_maintenance8", true)
		DeviceFeature("Maintenance8", true)

		DeviceExtension("VK_KHR_maintenance9", true)
		DeviceFeature("Maintenance9", true)

		DeviceExtension("VK_KHR_calibrated_timestamps", true)

		DeviceExtension("VK_EXT_shader_object", true)
		DeviceFeature("ShaderObject", true)

		DeviceExtension("VK_EXT_image_view_min_lod", false)
		DeviceFeature("MinLod", false)

		// TODO: gate on whether we have these features
		DeviceExtension("VK_EXT_mesh_shader", false)
		DeviceFeature("TaskShader", false)
		DeviceFeature("MeshShader", false)

		DeviceExtension("VK_KHR_deferred_host_operations", false)

		DeviceExtension("VK_KHR_acceleration_structure", false)
		DeviceFeature("AccelerationStructure", false)

		DeviceExtension("VK_KHR_pipeline_library", false)

		DeviceExtension("VK_EXT_pipeline_library_group_handles", false)
		DeviceFeature("PipelineLibraryGroupHandles", false)

		DeviceExtension("VK_KHR_ray_tracing_pipeline", false)
		DeviceFeature("RayTracingPipeline", false)

		DeviceExtension("VK_KHR_ray_query", false)
		DeviceFeature("RayQuery", false)

		DeviceExtension("VK_KHR_ray_tracing_maintenance1", false)
		DeviceFeature("RayTracingMaintenance1", false)

		DeviceExtension("VK_KHR_ray_tracing_position_fetch", false)
		DeviceFeature("RayTracingPositionFetch", false)

		// DeviceExtension("VK_EXT_external_memory_host", false)

		DeviceExtension("VK_KHR_swapchain", false)
		DeviceExtension("VK_KHR_swapchain_mutable_format", false)

		var availableFeatures features
		availableFeatures.init(false)
		pinner.Pin(&availableFeatures)
		vkFns.GetPhysicalDeviceFeatures2(physicalDevice, &availableFeatures.PhysicalDeviceFeatures2)

		var enabledFeatures features
		// TODO: subset things to just the available features
		for _, feature := range slices.Sorted(maps.Keys(requestedDeviceFeatures)) {
			if !availableFeatures.Get(feature) {
				if requestedDeviceFeatures[feature] {
					panic("don't have required feature " + feature)
				}
				continue
			}
			enabledFeatures.Set(feature)
		}
		enabledFeatures.init(true)

		queueFamilies = defaultQueues()

		// println(fmt.Sprintf("queues %d order %d", queueCounts, queueFamilies.probe))

		queueCreateInfos := slices.Collect(func(yield func(vk.DeviceQueueCreateInfo) bool) {
			for family, count := range queueFamilies.counts {
				if count == 0 {
					continue
				}
				prios := make([]float32, count)
				yield(vk.DeviceQueueCreateInfo{
					SType:            vk.STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO,
					QueueFamilyIndex: uint32(family),
					QueueCount:       uint32(count),
					PQueuePriorities: pinnedSliceData(&pinner, prios),
				})
			}
		})

		enabledDeviceExtensionsSlice := slices.Sorted(maps.Keys(requestedDeviceExtensions))

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

		for _, family := range queueFamilies.All() {
			for i := range queueFamilies.props[family].QueueCount {
				newSubmissionQueue(family, i)
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
		}, nil, &descriptorSetLayout))

		must(vkFns.CreatePipelineLayout(device, &vk.PipelineLayoutCreateInfo{
			SType:                  vk.STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
			SetLayoutCount:         1,
			PSetLayouts:            pinned(&pinner, &descriptorSetLayout),
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
			PSetLayouts:        pinned(&pinner, &descriptorSetLayout),
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
