package gpu

import (
	"reflect"

	"worldspawn/gpu/vk"
)

// TODO: autogenerate this
type deviceFeatures struct {
	vk.PhysicalDeviceFeatures2
	vk.PhysicalDeviceVulkan11Features
	vk.PhysicalDeviceVulkan12Features
	vk.PhysicalDeviceVulkan13Features
	vk.PhysicalDeviceVulkan14Features
	vk.PhysicalDeviceMaintenance7FeaturesKHR
	vk.PhysicalDeviceMaintenance8FeaturesKHR
	vk.PhysicalDeviceMaintenance9FeaturesKHR
	vk.PhysicalDeviceDeviceAddressCommandsFeaturesKHR
	vk.PhysicalDeviceInternallySynchronizedQueuesFeaturesKHR
	vk.PhysicalDeviceDescriptorHeapFeaturesEXT
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

func (features *deviceFeatures) init(skipzero bool) {
	rfeatures := reflect.ValueOf(features).Elem()

	for i, j := 0, -1; i < rfeatures.Type().NumField(); i++ {
		f := rfeatures.Field(i)
		if skipzero && f.IsZero() {
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

func (features *deviceFeatures) Get(name string) bool {
	rv := reflect.ValueOf(features).Elem().FieldByName(name)
	return rv.IsValid() && *rv.Addr().Interface().(*vk.Bool32) != vk.FALSE
}

func (features *deviceFeatures) Set(name string) {
	rv := reflect.ValueOf(features).Elem().FieldByName(name)
	*rv.Addr().Interface().(*vk.Bool32) = vk.TRUE
}
