package gpu

import (
	"fmt"
	"runtime"
	"slices"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: helper for building an SBT?

// TODO: add specifying the payload and hit attrib sizes i.e. pipe lib interface

// TODO: switch to dynamic stack

type RayTracingLibraryInterface struct {
	MaxRayPayloadSize      int
	MaxRayHitAttributeSize int
}

const maxPipelineRayPayloadSize = 32
const maxPipelineRayHitAttributeSize = 32

type RayTracingShaderGroup struct {
	vk     vk.Pipeline
	handle [32]byte
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
	if err := vkFns.CreateRayTracingPipelinesKHR(device, vk.NULL_HANDLE, vk.NULL_HANDLE, 1, &vk.RayTracingPipelineCreateInfoKHR{
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
	}, nil, &vkPipeline); err != nil {
		panic(fmt.Sprintf("gpu: vkCreateRayTracingPipelinesKHR: %v", err))
	}

	var handle [32]byte
	if err := vkFns.GetRayTracingShaderGroupHandlesKHR(device, vkPipeline, 0, 1, int(len(handle)), unsafe.Pointer(&handle)); err != nil {
		panic(fmt.Sprintf("gpu: vkGetRayTracingShaderGroupHandlesKHR: %v", err))
	}

	return &RayTracingShaderGroup{vk: vkPipeline, handle: handle}
}

func (sg *RayTracingShaderGroup) Handle() []byte {
	return sg.handle[:]
}

type RayTracingPipeline struct {
	vk vk.Pipeline
}

func LinkRayTracingShaderGroups(shaderGroups []*RayTracingShaderGroup) *RayTracingPipeline {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	libs := make([]vk.Pipeline, len(shaderGroups))
	for i, g := range shaderGroups {
		libs[i] = g.vk
	}

	var vkPipeline vk.Pipeline
	if err := vkFns.CreateRayTracingPipelinesKHR(device, vk.NULL_HANDLE, vk.NULL_HANDLE, 1, &vk.RayTracingPipelineCreateInfoKHR{
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
	}, nil, &vkPipeline); err != nil {
		panic(fmt.Sprintf("gpu: vkCreateRayTracingPipelinesKHR: %v", err))
	}

	return &RayTracingPipeline{vk: vkPipeline}
}

// TODO: use shorter names here. Replace ShaderBindingTable in the field names
// to ShaderRecords or just Records.
// TODO: make this strongly/er typed by introducing the type parameters for
// record types? (just the shader data part, skipping group handle)
type ShaderBindingTable struct {
	RaygenShaderRecordAddress         UnsafePointer
	RaygenShaderRecordSize            int
	MissShaderBindingTableAddress     UnsafePointer
	MissShaderBindingTableSize        int
	MissShaderBindingTableStride      int
	HitShaderBindingTableAddress      UnsafePointer
	HitShaderBindingTableSize         int
	HitShaderBindingTableStride       int
	CallableShaderBindingTableAddress UnsafePointer
	CallableShaderBindingTableSize    int
	CallableShaderBindingTableStride  int
}

type traceRaysJob struct {
	pipeline *RayTracingPipeline
	sbt      ShaderBindingTable
	width    uint32
	height   uint32
	depth    uint32
	args     []byte
}

func EnqueueTraceRays(jq *JobQueue,
	pipeline *RayTracingPipeline,
	sbt *ShaderBindingTable,
	width, height, depth int,
	args any) {
	jq.Enqueue(&traceRaysJob{
		pipeline: pipeline,
		sbt:      *sbt,
		width:    uint32(width),
		height:   uint32(height),
		depth:    uint32(depth),
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

		vkFns.CmdBindPipeline(cb, vk.PIPELINE_BIND_POINT_RAY_TRACING_KHR, job.pipeline.vk)

		vkFns.CmdBindDescriptorSets(
			cb,
			vk.PIPELINE_BIND_POINT_RAY_TRACING_KHR,
			pipelineLayout,
			0,
			1, &descriptorSet,
			0, nil)

		pinner.Pin(unsafe.SliceData(job.args))
		vkFns.CmdPushConstants(
			cb,
			pipelineLayout,
			vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
			0,
			uint32(len(job.args)), unsafe.Pointer(unsafe.SliceData(job.args)))

		vkFns.CmdTraceRaysKHR(cb,
			&vk.StridedDeviceAddressRegionKHR{
				DeviceAddress: vk.DeviceAddress(job.sbt.RaygenShaderRecordAddress),
				Stride:        vk.DeviceSize(job.sbt.RaygenShaderRecordSize),
				Size:          vk.DeviceSize(job.sbt.RaygenShaderRecordSize),
			},
			&vk.StridedDeviceAddressRegionKHR{
				DeviceAddress: vk.DeviceAddress(job.sbt.MissShaderBindingTableAddress),
				Stride:        vk.DeviceSize(job.sbt.MissShaderBindingTableStride),
				Size:          vk.DeviceSize(job.sbt.MissShaderBindingTableSize),
			},
			&vk.StridedDeviceAddressRegionKHR{
				DeviceAddress: vk.DeviceAddress(job.sbt.HitShaderBindingTableAddress),
				Stride:        vk.DeviceSize(job.sbt.HitShaderBindingTableStride),
				Size:          vk.DeviceSize(job.sbt.HitShaderBindingTableSize),
			},
			&vk.StridedDeviceAddressRegionKHR{
				DeviceAddress: vk.DeviceAddress(job.sbt.CallableShaderBindingTableAddress),
				Stride:        vk.DeviceSize(job.sbt.CallableShaderBindingTableStride),
				Size:          vk.DeviceSize(job.sbt.CallableShaderBindingTableSize),
			},
			job.width,
			job.height,
			job.depth)
	})
}
