package rendering

import (
	"runtime"
	"unsafe"

	"worldspawn/gpu"
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

func NewFunc(blob []byte, stage vk.ShaderStageFlagBits, entry string) *Func {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	// TODO: outline the cases into separate functions and clean this up

	switch stage {
	case vk.SHADER_STAGE_VERTEX_BIT,
		vk.SHADER_STAGE_FRAGMENT_BIT,
		vk.SHADER_STAGE_COMPUTE_BIT:
		var vkShader vk.ShaderEXT
		must(vkFns.CreateShadersEXT(device, 1, &vk.ShaderCreateInfoEXT{
			SType:                  vk.STRUCTURE_TYPE_SHADER_CREATE_INFO_EXT,
			Stage:                  stage,
			NextStage:              nextStages(stage),
			CodeType:               vk.SHADER_CODE_TYPE_SPIRV_EXT,
			CodeSize:               len(blob),
			PCode:                  unsafe.Pointer(pinnedSliceData(&pinner, blob)),
			PName:                  pinnedCString(&pinner, entry),
			SetLayoutCount:         1,
			PSetLayouts:            pinned(&pinner, &gpu.DescriptorSetLayout),
			PushConstantRangeCount: 1,
			PPushConstantRanges: pinned(&pinner, &vk.PushConstantRange{
				StageFlags: vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
				Offset:     0,
				Size:       maxShaderArgsSize,
			}),
		}, nil, &vkShader))
		return &Func{stage: stage, vk: uint64(vkShader), entry: entry}

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

func nextStages(stage vk.ShaderStageFlagBits) vk.ShaderStageFlags {
	switch stage {
	case vk.SHADER_STAGE_VERTEX_BIT:
		return vk.ShaderStageFlags(vk.SHADER_STAGE_FRAGMENT_BIT)

	case vk.SHADER_STAGE_FRAGMENT_BIT:
		return 0

	case vk.SHADER_STAGE_COMPUTE_BIT:
		return 0

	default:
		panic("unreachable")
	}
}
