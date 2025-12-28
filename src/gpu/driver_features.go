package gpu

import (
	"reflect"

	"worldspawn/gpu/vk"
)

// TODO: generate features and properties to include all of the features? Also
// make this public if we intend to allow external packages to inject stuff.

type features struct {
	Vulkan10                    vk.PhysicalDeviceFeatures2
	Vulkan11                    vk.PhysicalDeviceVulkan11Features
	Vulkan12                    vk.PhysicalDeviceVulkan12Features
	Vulkan13                    vk.PhysicalDeviceVulkan13Features
	Vulkan14                    vk.PhysicalDeviceVulkan14Features
	Maintenance7                vk.PhysicalDeviceMaintenance7FeaturesKHR
	Maintenance8                vk.PhysicalDeviceMaintenance8FeaturesKHR
	ShaderObject                vk.PhysicalDeviceShaderObjectFeaturesEXT
	MeshShader                  vk.PhysicalDeviceMeshShaderFeaturesEXT
	ImageViewMinLod             vk.PhysicalDeviceImageViewMinLodFeaturesEXT
	AccelerationStructure       vk.PhysicalDeviceAccelerationStructureFeaturesKHR
	PipelineLibraryGroupHandles vk.PhysicalDevicePipelineLibraryGroupHandlesFeaturesEXT
	RayTracingPipeline          vk.PhysicalDeviceRayTracingPipelineFeaturesKHR
	RayQuery                    vk.PhysicalDeviceRayQueryFeaturesKHR
	RayTracingMaintenance1      vk.PhysicalDeviceRayTracingMaintenance1FeaturesKHR
	RayTracingPositionFetch     vk.PhysicalDeviceRayTracingPositionFetchFeaturesKHR
}

func (features *features) init() {
	rfeatures := reflect.ValueOf(features).Elem()
	for i, j := 0, -1; i < rfeatures.Type().NumField(); i++ {
		f := rfeatures.Field(i)
		f.Field(1 /* .SType */).SetInt(int64(vk.SType(f.Interface())))
		if j >= 0 {
			g := rfeatures.Field(j)
			g.Field(2 /* .Next */).SetPointer(f.Addr().UnsafePointer())
		}
		j = i
	}
}

func (features *features) prepareForDeviceCreate() {
	rfeatures := reflect.ValueOf(features).Elem()
	for i, j := 0, -1; i < rfeatures.Type().NumField(); i++ {
		f := rfeatures.Field(i)
		f.Field(1 /* .SType */).SetInt(int64(vk.SType(f.Interface())))
		tmp := reflect.New(f.Type()).Elem()
		tmp.Set(f)
		tmp.Field(1 /* .SType */).SetZero()
		tmp.Field(2 /* .Next */).SetZero()
		if tmp.IsZero() {
			continue
		}
		if j >= 0 {
			g := rfeatures.Field(j)
			g.Field(2 /* .Next */).SetPointer(f.Addr().UnsafePointer())
		}
		j = i
	}
}

type properties struct {
	Vulkan10           vk.PhysicalDeviceProperties2
	Vulkan12           vk.PhysicalDeviceVulkan12Properties
	MeshShader         vk.PhysicalDeviceMeshShaderPropertiesEXT
	RayTracingPipeline vk.PhysicalDeviceRayTracingPipelinePropertiesKHR
}

func (props *properties) init() {
	rprops := reflect.ValueOf(props).Elem()
	for i, j := 0, -1; i < rprops.Type().NumField(); i++ {
		f := rprops.Field(i)
		f.Field(1 /* .SType */).SetInt(int64(vk.SType(f.Interface())))
		if j >= 0 {
			g := rprops.Field(j)
			g.Field(2 /* .Next */).SetPointer(f.Addr().UnsafePointer())
		}
		j = i
	}
}
