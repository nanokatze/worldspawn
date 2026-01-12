package rendering

import (
	"runtime"
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

// TODO: should use gpu's definition
const maxShaderArgsSize = 256

// TODO: stronger typing? i.e. type per stage.
// TODO: rename back to Func or whatever
type Shader struct {
	vk vk.ShaderEXT

	stage vk.ShaderStageFlagBits
	entry string
}

// TODO: optimized shader linking with state, fed by profiling?

func NewShader(blob []byte, stage vk.ShaderStageFlagBits, entry string) *Shader {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	// TODO: validate stage. I guess we could also infer it from the []byte?

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
	return &Shader{stage: stage, vk: vkShader, entry: entry}
}

func (f *Shader) String() string {
	// TODO: print more stuff like the file
	return f.entry
}

func (f *Shader) vkShader() vk.ShaderEXT {
	if f == nil {
		return vk.NULL_HANDLE
	}
	return f.vk
}

func nextStages(stage vk.ShaderStageFlagBits) vk.ShaderStageFlags {
	switch stage {
	case vk.SHADER_STAGE_VERTEX_BIT:
		return vk.ShaderStageFlags(vk.SHADER_STAGE_FRAGMENT_BIT)

	case vk.SHADER_STAGE_FRAGMENT_BIT:
		return 0

	default:
		panic("unreachable")
	}
}
