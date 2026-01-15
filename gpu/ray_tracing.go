package gpu

import (
	"runtime"
	"slices"
	"structs"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: helper for building an SBT?

// TODO: add specifying the payload and hit attrib sizes i.e. pipe lib interface

// TODO: switch to dynamic stack

// TODO: put this into shader.go?
type RayTracingLibraryInterface struct {
	MaxRayPayloadSize      int
	MaxRayHitAttributeSize int
}

const maxPipelineRayPayloadSize = 32
const maxPipelineRayHitAttributeSize = 32

type RayTracingShaderGroup struct {
	vk     vk.Pipeline
	handle RayTracingShaderGroupHandle
}

func NewGeneralRayTracingShaderGroup(general *Func) *RayTracingShaderGroup {
	return newRayTracingShaderGroup(
		vk.RAY_TRACING_SHADER_GROUP_TYPE_GENERAL_KHR,
		general, nil, nil, nil)
}

func NewTrianglesRayTracingShaderGroup(closestHit, anyHit *Func) *RayTracingShaderGroup {
	return newRayTracingShaderGroup(
		vk.RAY_TRACING_SHADER_GROUP_TYPE_TRIANGLES_HIT_GROUP_KHR,
		nil, closestHit, anyHit, nil)
}

// TODO: return an RT pipe + shader group instead?
func newRayTracingShaderGroup(_type vk.RayTracingShaderGroupTypeKHR,
	general, closestHit, anyHit, intersection *Func) *RayTracingShaderGroup {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	group := &vk.RayTracingShaderGroupCreateInfoKHR{
		SType:              vk.STRUCTURE_TYPE_RAY_TRACING_SHADER_GROUP_CREATE_INFO_KHR,
		Type:               _type,
		GeneralShader:      ^uint32(0),
		ClosestHitShader:   ^uint32(0),
		AnyHitShader:       ^uint32(0),
		IntersectionShader: ^uint32(0),
	}

	var libs []vk.Pipeline

	if general != nil {
		libs = append(libs, general.vkPipeline())
		group.GeneralShader = uint32(len(libs) - 1)
	}
	if closestHit != nil {
		libs = append(libs, closestHit.vkPipeline())
		group.ClosestHitShader = uint32(len(libs) - 1)
	}
	if anyHit != nil {
		libs = append(libs, anyHit.vkPipeline())
		group.AnyHitShader = uint32(len(libs) - 1)
	}
	if intersection != nil {
		libs = append(libs, intersection.vkPipeline())
		group.IntersectionShader = uint32(len(libs) - 1)
	}

	var vkPipeline vk.Pipeline
	must(vkFns.CreateRayTracingPipelinesKHR(device, vk.NULL_HANDLE, vk.NULL_HANDLE, 1, &vk.RayTracingPipelineCreateInfoKHR{
		SType:      vk.STRUCTURE_TYPE_RAY_TRACING_PIPELINE_CREATE_INFO_KHR,
		Flags:      vk.PipelineCreateFlags(vk.PIPELINE_CREATE_LIBRARY_BIT_KHR),
		GroupCount: 1,
		PGroups:    pinned(&pinner, group),
		PLibraryInfo: pinned(&pinner, &vk.PipelineLibraryCreateInfoKHR{
			SType:        vk.STRUCTURE_TYPE_PIPELINE_LIBRARY_CREATE_INFO_KHR,
			LibraryCount: uint32(len(libs)),
			PLibraries:   unsafe.SliceData(libs),
		}),
		PLibraryInterface: pinned(&pinner, &vk.RayTracingPipelineInterfaceCreateInfoKHR{
			SType:                          vk.STRUCTURE_TYPE_RAY_TRACING_PIPELINE_INTERFACE_CREATE_INFO_KHR,
			MaxPipelineRayPayloadSize:      maxPipelineRayPayloadSize,
			MaxPipelineRayHitAttributeSize: maxPipelineRayHitAttributeSize,
		}),
		Layout:                       pipelineLayout,
		MaxPipelineRayRecursionDepth: 1, // part of the lib interface
	}, nil, &vkPipeline))

	var handle RayTracingShaderGroupHandle
	must(vkFns.GetRayTracingShaderGroupHandlesKHR(device, vkPipeline, 0, 1, int(unsafe.Sizeof(handle)), unsafe.Pointer(&handle)))

	return &RayTracingShaderGroup{vk: vkPipeline, handle: handle}
}

// TODO: rename
type RayTracingShaderGroupHandle struct {
	_ structs.HostLayout
	// TODO: when https://github.com/golang/go/issues/19057 is out, force align
	// this struct to 32
	h [32]byte
}

func (sg *RayTracingShaderGroup) Handle() RayTracingShaderGroupHandle {
	return sg.handle
}

type RayTracingPipeline struct {
	vk vk.Pipeline
}

func (pipeline *RayTracingPipeline) Destroy() {
	panic("not implemented")
}

// TODO: make it possible to link RT pipes into more RT pipes
func LinkRayTracingShaderGroups(shaderGroups ...*RayTracingShaderGroup) *RayTracingPipeline {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	libs := make([]vk.Pipeline, len(shaderGroups))
	for i, g := range shaderGroups {
		libs[i] = g.vk
	}

	var vkPipeline vk.Pipeline
	must(vkFns.CreateRayTracingPipelinesKHR(device, vk.NULL_HANDLE, vk.NULL_HANDLE, 1, &vk.RayTracingPipelineCreateInfoKHR{
		SType: vk.STRUCTURE_TYPE_RAY_TRACING_PIPELINE_CREATE_INFO_KHR,
		PLibraryInfo: pinned(&pinner, &vk.PipelineLibraryCreateInfoKHR{
			SType:        vk.STRUCTURE_TYPE_PIPELINE_LIBRARY_CREATE_INFO_KHR,
			LibraryCount: uint32(len(libs)),
			PLibraries:   unsafe.SliceData(libs),
		}),
		PLibraryInterface: pinned(&pinner, &vk.RayTracingPipelineInterfaceCreateInfoKHR{
			SType:                          vk.STRUCTURE_TYPE_RAY_TRACING_PIPELINE_INTERFACE_CREATE_INFO_KHR,
			MaxPipelineRayPayloadSize:      maxPipelineRayPayloadSize,
			MaxPipelineRayHitAttributeSize: maxPipelineRayHitAttributeSize,
		}),
		Layout:                       pipelineLayout,
		MaxPipelineRayRecursionDepth: 1, // this would have to be user-provided
	}, nil, &vkPipeline))

	return &RayTracingPipeline{vk: vkPipeline}
}

type shaderRecordSlice struct {
	_       structs.HostLayout
	address UnsafePointer
	size    int
	stride  int
}

func makeShaderRecordSlice[T any](s Slice[T]) shaderRecordSlice {
	elemSize := int(unsafe.Sizeof(s.Value()[0]))
	return shaderRecordSlice{
		address: UnsafePointer(SliceData(s)),
		size:    SliceLen(s) * elemSize,
		stride:  elemSize,
	}
}

// TODO: rename?
type ShaderBindingTable struct {
	_                   structs.HostLayout
	raygenRecordAddress UnsafePointer // TODO: put these into a struct too?
	raygenRecordSize    int
	missRecords         shaderRecordSlice
	hitRecords          shaderRecordSlice
	callableRecords     shaderRecordSlice
}

func MakeShaderBindingTable[A, B, C, D any](raygenRecord Pointer[A], missRecords Slice[B], hitRecords Slice[C], callableRecords Slice[D]) ShaderBindingTable {
	// TODO: validate that the types are suitable (i.e. their sizes are
	// multiples of 32) and the resulting sbt is valid
	return ShaderBindingTable{
		raygenRecordAddress: UnsafePointer(raygenRecord),
		raygenRecordSize:    int(unsafe.Sizeof(raygenRecord.Value())),
		missRecords:         makeShaderRecordSlice(missRecords),
		hitRecords:          makeShaderRecordSlice(hitRecords),
		callableRecords:     makeShaderRecordSlice(callableRecords),
	}
}

type traceRaysJob struct {
	threads  [3]uint32
	pipeline *RayTracingPipeline
	sbt      ShaderBindingTable
	args     []byte
}

func EnqueueTraceRays(
	jq *JobQueue,
	threads []int,
	pipeline *RayTracingPipeline,
	sbt ShaderBindingTable,
	args any) {
	if err := validateDispatchGrid2(threads); err != nil {
		if err == errEmptyGrid {
			return
		}
		panic(err)
	}

	threads32 := [3]uint32{1, 1, 1}
	for i, d := range threads {
		threads32[i] = uint32(d)
	}

	jq.Enqueue(&traceRaysJob{
		threads:  threads32,
		pipeline: pipeline,
		sbt:      sbt,
		args:     slices.Clone(asbytes(args)),
	})
}

func (*traceRaysJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: queueFamilies.Mask(0b010),
	}
}

func (job *traceRaysJob) Exec(q *CommandQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		BindDescriptorSet(cb, vk.PIPELINE_BIND_POINT_RAY_TRACING_KHR)

		vkFns.CmdBindPipeline(cb, vk.PIPELINE_BIND_POINT_RAY_TRACING_KHR, job.pipeline.vk)

		pinner.Pin(unsafe.SliceData(job.args))
		vkFns.CmdPushConstants(
			cb,
			pipelineLayout,
			vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
			0,
			uint32(len(job.args)), unsafe.Pointer(unsafe.SliceData(job.args)))

		vkFns.CmdTraceRaysKHR(cb,
			&vk.StridedDeviceAddressRegionKHR{
				DeviceAddress: vk.DeviceAddress(job.sbt.raygenRecordAddress),
				Stride:        vk.DeviceSize(job.sbt.raygenRecordSize),
				Size:          vk.DeviceSize(job.sbt.raygenRecordSize),
			},
			&vk.StridedDeviceAddressRegionKHR{
				DeviceAddress: vk.DeviceAddress(job.sbt.missRecords.address),
				Stride:        vk.DeviceSize(job.sbt.missRecords.stride),
				Size:          vk.DeviceSize(job.sbt.missRecords.size),
			},
			&vk.StridedDeviceAddressRegionKHR{
				DeviceAddress: vk.DeviceAddress(job.sbt.hitRecords.address),
				Stride:        vk.DeviceSize(job.sbt.hitRecords.stride),
				Size:          vk.DeviceSize(job.sbt.hitRecords.size),
			},
			&vk.StridedDeviceAddressRegionKHR{
				DeviceAddress: vk.DeviceAddress(job.sbt.callableRecords.address),
				Stride:        vk.DeviceSize(job.sbt.callableRecords.stride),
				Size:          vk.DeviceSize(job.sbt.callableRecords.size),
			},
			job.threads[0],
			job.threads[1],
			job.threads[2])
	})
}
