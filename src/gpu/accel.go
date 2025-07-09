package gpu

import (
	"fmt"
	"runtime"
	"structs"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: use shorter names

type AccelBuildInput interface {
	vk(geometry *vk.AccelerationStructureGeometryKHR, primitiveCount *uint32)
}

type AccelBuildInputTriangles struct {
	VertexFormat   Format
	VertexBuffer   UnsafePointer // ignored by AccelerationStructureBuildConfig.Sizes
	VertexStride   int
	MaxVertex      uint32
	IndexType      IndexType
	IndexBuffer    UnsafePointer // ignored by AccelerationStructureBuildConfig.Sizes
	PrimitiveCount uint32
	Transform      Pointer[[3][4]float32] // checked for nil by AccelerationStructureBuildConfig.Sizes
}

func (triangles *AccelBuildInputTriangles) vk(geometry *vk.AccelerationStructureGeometryKHR, primitiveCount *uint32) {
	*geometry = vk.AccelerationStructureGeometryKHR{
		SType:        vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_KHR,
		GeometryType: vk.GEOMETRY_TYPE_TRIANGLES_KHR,
		Geometry: vk.AccelerationStructureGeometryDataTriangles(
			vk.AccelerationStructureGeometryTrianglesDataKHR{
				SType:         vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_TRIANGLES_DATA_KHR,
				VertexFormat:  triangles.VertexFormat,
				VertexData:    vk.DeviceOrHostAddressConstKHR(triangles.VertexBuffer),
				VertexStride:  vk.DeviceSize(triangles.VertexStride),
				MaxVertex:     triangles.MaxVertex,
				IndexType:     triangles.IndexType,
				IndexData:     vk.DeviceOrHostAddressConstKHR(triangles.IndexBuffer),
				TransformData: vk.DeviceOrHostAddressConstKHR(triangles.Transform),
			}),
	}
	*primitiveCount = triangles.PrimitiveCount
}

type AccelInstance struct {
	_                 structs.HostLayout
	Transform         [3][4]float32
	InstanceIDAndMask uint32 // TODO: make packing more user-friendly
	SBTOffsetAndFlags uint32
	Accel             UnsafePointer
}

type AccelBuildInputInstances struct {
	Instances     Pointer[AccelInstance] // ignored by AccelBuildConfig.CalcSizes
	InstanceCount uint32
}

func (instances *AccelBuildInputInstances) vk(geometry *vk.AccelerationStructureGeometryKHR, primitiveCount *uint32) {
	*geometry = vk.AccelerationStructureGeometryKHR{
		SType:        vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_KHR,
		GeometryType: vk.GEOMETRY_TYPE_INSTANCES_KHR,
		Geometry: vk.AccelerationStructureGeometryDataInstances(
			vk.AccelerationStructureGeometryInstancesDataKHR{
				SType:           vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_INSTANCES_DATA_KHR,
				ArrayOfPointers: vkBool32(false),
				Data:            vk.DeviceOrHostAddressConstKHR(instances.Instances),
			}),
	}
	*primitiveCount = instances.InstanceCount
}

type AccelBuildInputInstancePointers struct {
	Instances     Pointer[Pointer[AccelInstance]] // ignored by AccelBuildConfig.CalcSizes
	InstanceCount uint32
}

func (instancePointers *AccelBuildInputInstancePointers) vk(geometry *vk.AccelerationStructureGeometryKHR, primitiveCount *uint32) {
	*geometry = vk.AccelerationStructureGeometryKHR{
		SType:        vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_KHR,
		GeometryType: vk.GEOMETRY_TYPE_INSTANCES_KHR,
		Geometry: vk.AccelerationStructureGeometryDataInstances(
			vk.AccelerationStructureGeometryInstancesDataKHR{
				SType:           vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_INSTANCES_DATA_KHR,
				ArrayOfPointers: vkBool32(true),
				Data:            vk.DeviceOrHostAddressConstKHR(instancePointers.Instances),
			}),
	}
	*primitiveCount = instancePointers.InstanceCount
}

// TODO: rename, e.g. to AccelerationStructureBuildInputs?
type AccelBuildConfig struct {
	Type   vk.AccelerationStructureTypeKHR
	Inputs []AccelBuildInput
}

func (config *AccelBuildConfig) CalcSizes() (int, int, int) {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	gpuInit()

	vkGeometries := make([]vk.AccelerationStructureGeometryKHR, len(config.Inputs))
	vkMaxPrimitiveCounts := make([]uint32, len(config.Inputs))
	for i, input := range config.Inputs {
		input.vk(&vkGeometries[i], &vkMaxPrimitiveCounts[i])
	}
	pinner.Pin(unsafe.SliceData(vkGeometries))
	pinner.Pin(unsafe.SliceData(vkMaxPrimitiveCounts))

	sizes := vk.AccelerationStructureBuildSizesInfoKHR{
		SType: vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_BUILD_SIZES_INFO_KHR,
	}
	vkFns.GetAccelerationStructureBuildSizesKHR(
		device,
		vk.ACCELERATION_STRUCTURE_BUILD_TYPE_DEVICE_KHR,
		&vk.AccelerationStructureBuildGeometryInfoKHR{
			SType:         vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_BUILD_GEOMETRY_INFO_KHR,
			Type:          config.Type,
			GeometryCount: uint32(len(vkGeometries)),
			PGeometries:   unsafe.SliceData(vkGeometries),
		},
		unsafe.SliceData(vkMaxPrimitiveCounts),
		&sizes)

	return int(sizes.AccelerationStructureSize), int(sizes.BuildScratchSize), int(sizes.UpdateScratchSize)
}

type accelBuildJob struct {
	dst           UnsafePointer
	dstSize       int
	asType        vk.AccelerationStructureTypeKHR
	vkGeometries  []vk.AccelerationStructureGeometryKHR
	vkBuildRanges []vk.AccelerationStructureBuildRangeInfoKHR
	scratch       UnsafePointer
}

func EnqueueAccelBuild(jq *JobQueue, dst UnsafePointer, dstSize int, config *AccelBuildConfig, scratch UnsafePointer) {
	vkGeometries := make([]vk.AccelerationStructureGeometryKHR, len(config.Inputs))
	vkBuildRanges := make([]vk.AccelerationStructureBuildRangeInfoKHR, len(config.Inputs))
	for i, input := range config.Inputs {
		input.vk(&vkGeometries[i], &vkBuildRanges[i].PrimitiveCount)
	}

	jq.Enqueue(&accelBuildJob{
		dst:           dst,
		dstSize:       dstSize,
		asType:        config.Type,
		vkGeometries:  vkGeometries,
		vkBuildRanges: vkBuildRanges,
		scratch:       scratch,
	})
}

func (*accelBuildJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: queueFamilies.Mask(0b010),
	}
}

func (job *accelBuildJob) Exec(q *CommandQueue) {
	dstAS := newVkAccelerationStructureAt(job.dst, job.dstSize)

	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		pinner.Pin(unsafe.SliceData(job.vkGeometries))
		pinner.Pin(unsafe.SliceData(job.vkBuildRanges))

		pBuildRangeInfos := unsafe.SliceData(job.vkBuildRanges)
		ppBuildRangeInfos := &pBuildRangeInfos

		pinner.Pin(ppBuildRangeInfos)

		vkFns.CmdBuildAccelerationStructuresKHR(
			cb,
			1, &vk.AccelerationStructureBuildGeometryInfoKHR{
				SType:                    vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_BUILD_GEOMETRY_INFO_KHR,
				Type:                     job.asType,
				Mode:                     vk.BUILD_ACCELERATION_STRUCTURE_MODE_BUILD_KHR,
				DstAccelerationStructure: dstAS,
				GeometryCount:            uint32(len(job.vkGeometries)),
				PGeometries:              unsafe.SliceData(job.vkGeometries),
				ScratchData:              vk.DeviceOrHostAddressKHR(job.scratch),
			},
			ppBuildRangeInfos)
	})

	if true {
		q.Cleanup(func() { vkFns.DestroyAccelerationStructureKHR(device, dstAS, nil) })
	}
}

func newVkAccelerationStructureAt(address UnsafePointer, size int) vk.AccelerationStructureKHR {
	// !!!Massive API abuse ahead!!!

	if size == 0 {
		panic("size must not be 0")
	}

	// TODO: we could probably replace size with the remaining buffer size.

	buffer, offset := bufferAndOffset(address)

	var as vk.AccelerationStructureKHR
	if err := vkFns.CreateAccelerationStructureKHR(device, &vk.AccelerationStructureCreateInfoKHR{
		SType:  vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_CREATE_INFO_KHR,
		Buffer: buffer,
		Offset: offset,
		Size:   vk.DeviceSize(size),
		Type:   vk.ACCELERATION_STRUCTURE_TYPE_GENERIC_KHR,
	}, nil, &as); err != nil {
		panic(fmt.Sprintf("gpu: vkCreateAccelerationStructureKHR: %v", err))
	}

	asAddr := vkFns.GetAccelerationStructureDeviceAddressKHR(device, &vk.AccelerationStructureDeviceAddressInfoKHR{
		SType:                 vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_DEVICE_ADDRESS_INFO_KHR,
		AccelerationStructure: as,
	})
	if asAddr != vk.DeviceAddress(address) {
		panic("as addr does not match the addr as was created at")
	}

	return as
}
