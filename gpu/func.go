package gpu

import (
	"runtime"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: move elsewhere
const maxShaderArgsSize = 256

// TODO: split per stage. I.e. compute, graphics and ray tracing should be
// different types and live in their own... thing
type Func struct {
	stage vk.ShaderStageFlagBits
	vk    uint64 // vk.ShaderEXT or vk.Pipeline

	entry string
}

// TODO: optimized shader linking with state, fed by profiling

// TODO: RT pipe library interface

// TODO: rename to just NewFunc once we move stuff into ray tracing
func NewComputeFunc(blob []byte, entry string) *Func {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	gpuInit()

	var vkShader vk.ShaderEXT
	must(vkFns.CreateShadersEXT(device, 1, &vk.ShaderCreateInfoEXT{
		SType:                  vk.STRUCTURE_TYPE_SHADER_CREATE_INFO_EXT,
		Stage:                  vk.SHADER_STAGE_COMPUTE_BIT,
		CodeType:               vk.SHADER_CODE_TYPE_SPIRV_EXT,
		CodeSize:               len(blob),
		PCode:                  unsafe.Pointer(pinnedSliceData(&pinner, blob)),
		PName:                  pinnedCString(&pinner, entry),
		SetLayoutCount:         1,
		PSetLayouts:            pinned(&pinner, &DescriptorSetLayout),
		PushConstantRangeCount: 1,
		PPushConstantRanges: pinned(&pinner, &vk.PushConstantRange{
			StageFlags: vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
			Offset:     0,
			Size:       maxShaderArgsSize,
		}),
	}, nil, &vkShader))
	return &Func{stage: vk.SHADER_STAGE_COMPUTE_BIT, vk: uint64(vkShader), entry: entry}
}

func NewRayTracingFunc(blob []byte, stage vk.ShaderStageFlagBits, entry string) *Func {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	gpuInit()

	// TODO: outline the cases into separate functions and clean this up

	switch stage {
	case vk.SHADER_STAGE_RAYGEN_BIT_KHR,
		vk.SHADER_STAGE_ANY_HIT_BIT_KHR,
		vk.SHADER_STAGE_CLOSEST_HIT_BIT_KHR,
		vk.SHADER_STAGE_MISS_BIT_KHR,
		vk.SHADER_STAGE_INTERSECTION_BIT_KHR,
		vk.SHADER_STAGE_CALLABLE_BIT_KHR:
		var vkPipeline vk.Pipeline
		must(vkFns.CreateRayTracingPipelinesKHR(device, vk.NULL_HANDLE, vk.NULL_HANDLE, 1, &vk.RayTracingPipelineCreateInfoKHR{
			SType:      vk.STRUCTURE_TYPE_RAY_TRACING_PIPELINE_CREATE_INFO_KHR,
			Flags:      vk.PipelineCreateFlags(vk.PIPELINE_CREATE_LIBRARY_BIT_KHR),
			StageCount: uint32(1),
			PStages: pinned(&pinner, &vk.PipelineShaderStageCreateInfo{
				SType: vk.STRUCTURE_TYPE_PIPELINE_SHADER_STAGE_CREATE_INFO,
				PNext: unsafe.Pointer(pinned(&pinner, &vk.ShaderModuleCreateInfo{
					SType:    vk.STRUCTURE_TYPE_SHADER_MODULE_CREATE_INFO,
					CodeSize: len(blob),
					PCode:    (*uint32)(unsafe.Pointer(pinnedSliceData(&pinner, blob))),
				})),
				Stage: stage,
				PName: pinnedCString(&pinner, entry),
			}),
			PLibraryInterface: pinned(&pinner, &vk.RayTracingPipelineInterfaceCreateInfoKHR{
				SType:                          vk.STRUCTURE_TYPE_RAY_TRACING_PIPELINE_INTERFACE_CREATE_INFO_KHR,
				MaxPipelineRayPayloadSize:      maxPipelineRayPayloadSize,      // TODO: should be specified by the user
				MaxPipelineRayHitAttributeSize: maxPipelineRayHitAttributeSize, // TODO: should be specified by the user
			}),
			Layout:                       pipelineLayout,
			MaxPipelineRayRecursionDepth: 1, // part of the lib interface. Just make it dynamic
		}, nil, &vkPipeline))
		return &Func{stage: stage, vk: uint64(vkPipeline), entry: entry}

	default:
		panic("unsupported shader stage")
	}
}

func (f *Func) String() string {
	// TODO: print more stuff like the file
	return f.entry
}

func (f *Func) vkShader() vk.ShaderEXT {
	return vk.ShaderEXT(f.vk)
}

func (f *Func) vkPipeline() vk.Pipeline {
	return vk.Pipeline(f.vk)
}
