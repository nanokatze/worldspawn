package gpu

import (
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: make this private and replace with strongly typed wrappers (BLAS, TLAS)
type Accel struct {
	data UnsafePointer
	size int
}

func (accel Accel) Size() int { return accel.size }

type BLAS Accel

// TODO: rename to AccelGeometry?
type AccelBuildInput interface {
	vkAccelerationStructureGeometry(geometry *vk.AccelerationStructureGeometryKHR, primitiveCount *uint32)
}

// TODO: could we make VertexBuffer part somehow less annoying? E.g. with some
// kind of slice. Ok we can't use a slice here but we could provide a Set method
// (or Set methods) or a constructor. Or idk.
type AccelBuildInputTriangles struct {
	VertexFormat  vk.Format
	VertexBuffer  UnsafePointer // ignored by AccelBuildConfig.CalcSizes
	VertexCount   int
	VertexStride  int
	IndexType     vk.IndexType
	IndexBuffer   UnsafePointer // ignored by AccelBuildConfig.CalcSizes
	TriangleCount int
	Transform     Pointer[[3][4]float32] // checked for nil by AccelBuildConfig.CalcSizes
}

func (triangles *AccelBuildInputTriangles) vkAccelerationStructureGeometry(geometry *vk.AccelerationStructureGeometryKHR, primitiveCount *uint32) {
	*geometry = vk.AccelerationStructureGeometryKHR{
		SType:        vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_KHR,
		GeometryType: vk.GEOMETRY_TYPE_TRIANGLES_KHR,
		Geometry: vk.AccelerationStructureGeometryDataTriangles(
			vk.AccelerationStructureGeometryTrianglesDataKHR{
				SType:         vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_GEOMETRY_TRIANGLES_DATA_KHR,
				VertexFormat:  triangles.VertexFormat,
				VertexData:    vk.DeviceOrHostAddressConstKHR(triangles.VertexBuffer),
				VertexStride:  vk.DeviceSize(triangles.VertexStride),
				MaxVertex:     uint32(max(triangles.VertexCount-1, 0)),
				IndexType:     triangles.IndexType,
				IndexData:     vk.DeviceOrHostAddressConstKHR(triangles.IndexBuffer),
				TransformData: vk.DeviceOrHostAddressConstKHR(triangles.Transform),
			}),
	}
	*primitiveCount = uint32(triangles.TriangleCount)
}

// TODO: we can't replace these with Slices, otherwise CalcSizes wouldn't work.
// What we could do instead is make the fields here private and cook up
// constructors that take Slice and an ordinary int, or something of that kind,
// basically.
type AccelBuildInputInstances struct {
	Instances     Pointer[AccelInstance] // ignored by AccelBuildConfig.CalcSizes
	InstanceCount uint32
}

func (instances *AccelBuildInputInstances) vkAccelerationStructureGeometry(geometry *vk.AccelerationStructureGeometryKHR, primitiveCount *uint32) {
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

func (instancePointers *AccelBuildInputInstancePointers) vkAccelerationStructureGeometry(geometry *vk.AccelerationStructureGeometryKHR, primitiveCount *uint32) {
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

// TODO: split this for different accel types? CLAS consumes inputs in a very
// different shape and form.
type AccelBuildConfig struct {
	Type   vk.AccelerationStructureTypeKHR
	Inputs []AccelBuildInput
}

type AccelSizes struct {
	Accel         int
	BuildScratch  int
	UpdateScratch int
}

// TODO: should we split this into two, CalcAccelSize() and CalcScratchSizes()?
func (config *AccelBuildConfig) CalcSizes() AccelSizes {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	gpuInit()

	vkGeometries := make([]vk.AccelerationStructureGeometryKHR, len(config.Inputs))
	vkMaxPrimitiveCounts := make([]uint32, len(config.Inputs))
	for i, input := range config.Inputs {
		input.vkAccelerationStructureGeometry(&vkGeometries[i], &vkMaxPrimitiveCounts[i])
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

	return AccelSizes{
		Accel:         int(sizes.AccelerationStructureSize),
		BuildScratch:  int(sizes.BuildScratchSize),
		UpdateScratch: int(sizes.UpdateScratchSize),
	}
}

// TODO: NewAccelAt and helpers that take build configs
func NewAccel(size int) Accel {
	return Accel{
		data: UnsafePointer(SliceData(MakeSliceUncached[byte](size))),
		size: size,
	}
}

// TODO: rename to NewAccelTopLevel?
func NewTopLevelAccel(maxInstances int) Accel {
	config := &AccelBuildConfig{
		Type: vk.ACCELERATION_STRUCTURE_TYPE_TOP_LEVEL_KHR,
		Inputs: []AccelBuildInput{
			&AccelBuildInputInstances{
				InstanceCount: uint32(maxInstances),
			},
		},
	}
	return NewAccel(config.CalcSizes().Accel)
}

// TODO: change it to not use a pointer receiver?
func (config *AccelBuildConfig) EnqueueBuild(jq *JobQueue, accel Accel) {
	sizes := config.CalcSizes()
	if sizes.Accel > accel.size {
		panic("bad")
	}
	scratch := UnsafePointer(SliceData(MakeSliceUncached[byte](sizes.BuildScratch)))
	defer jq.Cleanup(func() { Free(scratch) })
	EnqueueAccelBuild(jq, accel.data, accel.size, config, scratch)
}

type accelBuildJob struct {
	dst           UnsafePointer
	dstSize       int
	asType        vk.AccelerationStructureTypeKHR
	vkGeometries  []vk.AccelerationStructureGeometryKHR
	vkBuildRanges []vk.AccelerationStructureBuildRangeInfoKHR
	scratch       UnsafePointer
}

// TODO: hide from public
func EnqueueAccelBuild(jq *JobQueue, dst UnsafePointer, dstSize int, config *AccelBuildConfig, scratch UnsafePointer) {
	vkGeometries := make([]vk.AccelerationStructureGeometryKHR, len(config.Inputs))
	vkBuildRanges := make([]vk.AccelerationStructureBuildRangeInfoKHR, len(config.Inputs))
	for i, input := range config.Inputs {
		input.vkAccelerationStructureGeometry(&vkGeometries[i], &vkBuildRanges[i].PrimitiveCount)
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
		QueueFamilies: topology.QueueFamilies(vk.QueueFlags(vk.QUEUE_COMPUTE_BIT)),
	}
}

func (job *accelBuildJob) Exec(q *DeviceQueue) {
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

	buffer, offset := BufferAndOffset(address)

	var as vk.AccelerationStructureKHR
	must(vkFns.CreateAccelerationStructureKHR(device, &vk.AccelerationStructureCreateInfoKHR{
		SType:  vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_CREATE_INFO_KHR,
		Buffer: buffer,
		Offset: offset,
		Size:   vk.DeviceSize(size),
		Type:   vk.ACCELERATION_STRUCTURE_TYPE_GENERIC_KHR,
	}, nil, &as))

	asAddr := vkFns.GetAccelerationStructureDeviceAddressKHR(device, &vk.AccelerationStructureDeviceAddressInfoKHR{
		SType:                 vk.STRUCTURE_TYPE_ACCELERATION_STRUCTURE_DEVICE_ADDRESS_INFO_KHR,
		AccelerationStructure: as,
	})
	if asAddr != vk.DeviceAddress(address) {
		panic("as addr does not match the addr as was created at")
	}

	return as
}
