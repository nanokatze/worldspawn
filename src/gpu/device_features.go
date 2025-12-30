package gpu

import (
	"reflect"

	"worldspawn/gpu/vk"
)

// TODO: autogenerate this
type features struct {
	vk.PhysicalDeviceFeatures2
	vk.PhysicalDeviceVulkan11Features
	vk.PhysicalDeviceVulkan12Features
	vk.PhysicalDeviceVulkan13Features
	vk.PhysicalDeviceVulkan14Features
	vk.PhysicalDeviceMaintenance7FeaturesKHR
	vk.PhysicalDeviceMaintenance8FeaturesKHR
	vk.PhysicalDeviceShaderObjectFeaturesEXT
	vk.PhysicalDeviceMeshShaderFeaturesEXT
	vk.PhysicalDeviceImageViewMinLodFeaturesEXT
	vk.PhysicalDeviceAccelerationStructureFeaturesKHR
	vk.PhysicalDevicePipelineLibraryGroupHandlesFeaturesEXT
	vk.PhysicalDeviceRayTracingPipelineFeaturesKHR
	vk.PhysicalDeviceRayQueryFeaturesKHR
	vk.PhysicalDeviceRayTracingMaintenance1FeaturesKHR
	vk.PhysicalDeviceRayTracingPositionFetchFeaturesKHR
}

var enabledDeviceExtensions = map[string]struct{}{}
var enabledDeviceFeatures = features{}

// TODO: make it a standalone method?
func (features *features) prepareForDeviceCreate() {
	rfeatures := reflect.ValueOf(features).Elem()

	for i, j := 0, -1; i < rfeatures.Type().NumField(); i++ {
		f := rfeatures.Field(i)
		if f.IsZero() {
			continue
		}

		f.Field(1 /* .SType */).SetInt(int64(vk.StructureTypeOf(f.Type())))

		if j >= 0 {
			g := rfeatures.Field(j)
			g.Field(2 /* .PNext */).SetPointer(f.Addr().UnsafePointer())
		}

		j = i
	}
}

// TODO: rename these to Require ...?

func EnableDeviceExtension(extension string) {
	enabledDeviceExtensions[extension] = struct{}{}
}

func EnableDeviceFeature(feature string) {
	reflect.ValueOf(&enabledDeviceFeatures).Elem().FieldByName(feature).SetUint(1)
}
