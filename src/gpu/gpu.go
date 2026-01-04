package gpu

import (
	"fmt"
	"log"
	"maps"
	"math/bits"
	"os"
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

// TODO: remove this and only leave the families []uint32 and vk.QueueFlags I
// guess. Other properties should be up to the other components.
type chosenQueueFamilies struct {
	props []vk.QueueFamilyProperties2
	probe []uint32
}

func chooseQueueFamilies(queueFamilyProps []vk.QueueFamilyProperties2) *chosenQueueFamilies {
	probe := chooseQueueFamilies2(queueFamilyProps)
	return &chosenQueueFamilies{
		props: queueFamilyProps,
		probe: probe,
	}
}

// Filters and sorts queues
func chooseQueueFamilies2(queueFamilyProps []vk.QueueFamilyProperties2) []uint32 {
	var families []uint32
	var visited [32]bool
	for _, wantQueueFlags := range []vk.QueueFlags{
		0b111,
		0b110,
		0b100,
	} {
		for family, props := range queueFamilyProps {
			if visited[family] {
				continue
			}
			if props.QueueFlags&wantQueueFlags != wantQueueFlags {
				continue
			}
			families = append(families, uint32(family))
			visited[family] = true
		}
	}
	slices.Reverse(families)
	return families
}

// TODO: lut for common single-bit masks
func (families *chosenQueueFamilies) Mask(wantQueueFlags vk.QueueFlags) uint32 {
	var mask uint32
	for i, props := range families.props {
		if props.QueueFlags&wantQueueFlags == wantQueueFlags {
			mask |= 1 << i
		}
	}
	return mask
}

// TODO: remove this
func (queueFamilies *chosenQueueFamilies) MinimumCapable(queueFlags vk.QueueFlags) int {
	return 32 - bits.LeadingZeros32(queueFamilies.Mask(queueFlags)) - 1
}

// TODO: remove this
func (families *chosenQueueFamilies) All() []uint32 {
	return families.probe
}

var instance vk.Instance
var physicalDevice vk.PhysicalDevice
var queueFamilies *chosenQueueFamilies
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
		if err != nil {
			panic(fmt.Sprintf("gpu: vkEnumerateInstanceLayerProperties: %v", err))
		}

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

		if err := vk.CreateInstance(pinned(&pinner, &vk.InstanceCreateInfo{
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
		}), nil, &instance); err != nil {
			panic(fmt.Sprintf("gpu: vkCreateInstance: %v", err))
		}

		vkFns.InstanceFuncs.Init(instance)

		physicalDevices, err := enumerate(func(len *uint32, data *vk.PhysicalDevice) error {
			return vkFns.EnumeratePhysicalDevices(instance, len, data)
		})
		if err != nil {
			panic(fmt.Sprintf("gpu: vkEnumeratePhysicalDevices: %v", err))
		}
		physicalDevice = physicalDevices[0]

		props := vk.PhysicalDeviceProperties2{
			SType: vk.STRUCTURE_TYPE_PHYSICAL_DEVICE_PROPERTIES_2,
		}
		vkFns.GetPhysicalDeviceProperties2(physicalDevice, &props)

		// TODO: we should use a user-provided function or logger to log stuff.
		// Actually let's not and just use println(stuff) gated behind an envvar?
		log.Print("GPU: ", byteSliceToString(props.DeviceName[:]))

		EnableDeviceFeature("SamplerAnisotropy")
		EnableDeviceFeature("ShaderInt64")
		EnableDeviceFeature("ShaderInt16")

		EnableDeviceFeature("StorageBuffer16BitAccess")
		EnableDeviceFeature("StoragePushConstant16")

		EnableDeviceFeature("StorageBuffer8BitAccess")
		EnableDeviceFeature("StoragePushConstant8")
		EnableDeviceFeature("ShaderFloat16")
		EnableDeviceFeature("ShaderInt8")
		EnableDeviceFeature("ShaderSampledImageArrayNonUniformIndexing")
		EnableDeviceFeature("ShaderStorageImageArrayNonUniformIndexing")
		EnableDeviceFeature("DescriptorBindingSampledImageUpdateAfterBind")
		EnableDeviceFeature("DescriptorBindingStorageImageUpdateAfterBind")
		EnableDeviceFeature("DescriptorBindingUpdateUnusedWhilePending")
		EnableDeviceFeature("DescriptorBindingPartiallyBound")
		EnableDeviceFeature("DescriptorBindingVariableDescriptorCount")
		EnableDeviceFeature("RuntimeDescriptorArray")
		EnableDeviceFeature("ScalarBlockLayout")
		EnableDeviceFeature("TimelineSemaphore")
		EnableDeviceFeature("BufferDeviceAddress")
		EnableDeviceFeature("VulkanMemoryModel")
		// Not sure we actually need these
		// EnableFeature("VulkanMemoryModelDeviceScope")
		// EnableFeature("VulkanMemoryModelAvailabilityVisibilityChains")

		// TODO: we also might need demote and discard, subgroup size control,
		// compute full subgroups
		EnableDeviceFeature("Synchronization2")
		EnableDeviceFeature("DynamicRendering")
		EnableDeviceFeature("Maintenance4")

		EnableDeviceFeature("Maintenance5")
		EnableDeviceFeature("Maintenance6")

		EnableDeviceExtension("VK_KHR_maintenance7")
		EnableDeviceFeature("Maintenance7")

		// TODO: enable maint{8,9} once we have them on deck
		if false {
			EnableDeviceExtension("VK_KHR_maintenance8")
			EnableDeviceFeature("Maintenance8")
		}

		EnableDeviceExtension("VK_KHR_calibrated_timestamps")

		EnableDeviceExtension("VK_EXT_shader_object")
		EnableDeviceFeature("ShaderObject")

		EnableDeviceExtension("VK_EXT_image_view_min_lod")
		EnableDeviceFeature("MinLod")

		// TODO: gate on whether we have these features
		if os.Getenv("SteamDeck") == "1" {
			if false {
				EnableDeviceExtension("VK_EXT_mesh_shader")
				EnableDeviceFeature("TaskShader")
				EnableDeviceFeature("MeshShader")
			}

			EnableDeviceExtension("VK_KHR_deferred_host_operations")

			EnableDeviceExtension("VK_KHR_acceleration_structure")
			EnableDeviceFeature("AccelerationStructure")

			EnableDeviceExtension("VK_KHR_pipeline_library")

			EnableDeviceExtension("VK_EXT_pipeline_library_group_handles")
			EnableDeviceFeature("PipelineLibraryGroupHandles")

			EnableDeviceExtension("VK_KHR_ray_tracing_pipeline")
			EnableDeviceFeature("RayTracingPipeline")

			EnableDeviceExtension("VK_KHR_ray_query")
			EnableDeviceFeature("RayQuery")

			EnableDeviceExtension("VK_KHR_ray_tracing_maintenance1")
			EnableDeviceFeature("RayTracingMaintenance1")

			EnableDeviceExtension("VK_KHR_ray_tracing_position_fetch")
			EnableDeviceFeature("RayTracingPositionFetch")
		}

		// EnableDeviceExtension("VK_EXT_external_memory_host")

		EnableDeviceExtension("VK_KHR_swapchain")
		EnableDeviceExtension("VK_KHR_swapchain_mutable_format")

		enabledDeviceFeatures.prepareForDeviceCreate()

		queueFamilyProps, _ := enumerate2(
			func(queueFamilyProps []vk.QueueFamilyProperties2) {
				for i := range queueFamilyProps {
					queueFamilyProps[i].SType = vk.STRUCTURE_TYPE_QUEUE_FAMILY_PROPERTIES_2
				}
			},
			func(len *uint32, data *vk.QueueFamilyProperties2) error {
				vkFns.GetPhysicalDeviceQueueFamilyProperties2(physicalDevice, len, data)
				return nil
			})

		queueFamilies = chooseQueueFamilies(queueFamilyProps)

		log.Printf("queue families %d", queueFamilies.probe)

		var queueCreateInfos []vk.DeviceQueueCreateInfo
		for _, family := range queueFamilies.All() {
			n := queueFamilyProps[family].QueueCount
			prios := make([]float32, n)
			queueCreateInfos = append(queueCreateInfos,
				vk.DeviceQueueCreateInfo{
					SType:            vk.STRUCTURE_TYPE_DEVICE_QUEUE_CREATE_INFO,
					QueueFamilyIndex: family,
					QueueCount:       n,
					PQueuePriorities: pinnedSliceData(&pinner, prios),
				})
		}

		enabledDeviceExtensionsSlice := slices.Sorted(maps.Keys(enabledDeviceExtensions))

		pinner.Pin(&enabledDeviceFeatures)

		if err := vkFns.CreateDevice(physicalDevice, &vk.DeviceCreateInfo{
			SType:                   vk.STRUCTURE_TYPE_DEVICE_CREATE_INFO,
			PNext:                   unsafe.Pointer(&enabledDeviceFeatures.PhysicalDeviceFeatures2),
			QueueCreateInfoCount:    uint32(len(queueCreateInfos)),
			PQueueCreateInfos:       pinnedSliceData(&pinner, queueCreateInfos),
			EnabledExtensionCount:   uint32(len(enabledDeviceExtensionsSlice)),
			PPEnabledExtensionNames: pinnedCStringSlice(&pinner, enabledDeviceExtensionsSlice),
		}, &device); err != nil {
			panic(fmt.Sprintf("gpu: vkCreateDevice: %v", err))
		}

		vkFns.DeviceFuncs.Init(device)

		for _, family := range queueFamilies.All() {
			for i := range queueFamilyProps[family].QueueCount {
				newSubmissionQueue(family, i)
			}
		}

		if err := vkFns.CreateDescriptorSetLayout(device, &vk.DescriptorSetLayoutCreateInfo{
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
		}, nil, &descriptorSetLayout); err != nil {
			panic(fmt.Sprintf("gpu: vkCreateDescriptorSetLayout: %v", err))
		}

		if err := vkFns.CreatePipelineLayout(device, &vk.PipelineLayoutCreateInfo{
			SType:                  vk.STRUCTURE_TYPE_PIPELINE_LAYOUT_CREATE_INFO,
			SetLayoutCount:         1,
			PSetLayouts:            pinned(&pinner, &descriptorSetLayout),
			PushConstantRangeCount: 1,
			PPushConstantRanges: pinned(&pinner, &vk.PushConstantRange{
				StageFlags: vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
				Offset:     0,
				Size:       maxShaderArgsSize,
			}),
		}, nil, &pipelineLayout); err != nil {
			panic(fmt.Sprintf("gpu: vkCreatePipelineLayout: %v", err))
		}

		var descriptorPool vk.DescriptorPool
		if err := vkFns.CreateDescriptorPool(device, &vk.DescriptorPoolCreateInfo{
			SType:         vk.STRUCTURE_TYPE_DESCRIPTOR_POOL_CREATE_INFO,
			Flags:         vk.DescriptorPoolCreateFlags(vk.DESCRIPTOR_POOL_CREATE_UPDATE_AFTER_BIND_BIT),
			MaxSets:       1,
			PoolSizeCount: 3,
			PPoolSizes: pinnedSliceData(&pinner, []vk.DescriptorPoolSize{
				{Type: vk.DESCRIPTOR_TYPE_SAMPLER, DescriptorCount: 2e3},
				{Type: vk.DESCRIPTOR_TYPE_SAMPLED_IMAGE, DescriptorCount: 1e6},
				{Type: vk.DESCRIPTOR_TYPE_STORAGE_IMAGE, DescriptorCount: 1e6},
			}),
		}, nil, &descriptorPool); err != nil {
			panic(fmt.Sprintf("gpu: vkCreateDescriptorPool: %v", err))
		}

		if err := vkFns.AllocateDescriptorSets(device, &vk.DescriptorSetAllocateInfo{
			SType:              vk.STRUCTURE_TYPE_DESCRIPTOR_SET_ALLOCATE_INFO,
			DescriptorPool:     descriptorPool,
			DescriptorSetCount: 1,
			PSetLayouts:        pinned(&pinner, &descriptorSetLayout),
		}, &descriptorSet); err != nil {
			panic(fmt.Sprintf("gpu: vkAllocateDescriptorSets: %v", err))
		}
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
