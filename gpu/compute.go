package gpu

import (
	"errors"
	"runtime"
	"slices"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: rename to make it clear that it has unbound env. ComputeClosureFunc?
// ComputeClosureBody?
type ComputeClosureBody[T any] struct {
	_  [0]T
	vk vk.ShaderEXT
}

// TODO: do something better pls
func CompileFunc[T any](blob []byte, entry string) ComputeClosureBody[T] {
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
	return ComputeClosureBody[T]{vk: vkShader}
}

func (body ComputeClosureBody[T]) WithEnv(env T) ComputeClosure[T] {
	return ComputeClosure[T]{
		Body: body,
		Env:  env,
	}
}

type ComputeClosure[T any] struct {
	Body ComputeClosureBody[T]
	Env  T
}

type dispatchJob struct {
	groups [3]uint32
	kernel vk.ShaderEXT
	args   []byte
}

// TODO: play with the order of arguments?
// TODO: should take an interface rather than be generic
func ParallelFor[T any](jq *JobQueue, groups []int, f ComputeClosure[T]) {
	if err := validateDispatchGrid2(groups); err != nil {
		if err == errEmptyGrid {
			return
		}
		panic(err)
	}

	groups32 := [3]uint32{1, 1, 1}
	for i, d := range groups {
		groups32[i] = uint32(d)
	}

	jq.Enqueue(&dispatchJob{
		groups: groups32,
		kernel: f.Body.vk,
		args:   slices.Clone(asbytes(&f.Env)),
	})
}

func (*dispatchJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: queueFamilies.Mask(vk.QueueFlags(vk.QUEUE_COMPUTE_BIT)),
	}
}

func (job *dispatchJob) Exec(q *CommandQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		BindDescriptorSet(cb, vk.PIPELINE_BIND_POINT_COMPUTE)

		vkFns.CmdBindShadersEXT(
			cb,
			1,
			unsafe.SliceData([]vk.ShaderStageFlagBits{vk.SHADER_STAGE_COMPUTE_BIT}),
			unsafe.SliceData([]vk.ShaderEXT{job.kernel}))

		vkFns.CmdPushConstants(
			cb,
			pipelineLayout,
			vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
			0,
			uint32(len(job.args)), unsafe.Pointer(unsafe.SliceData(job.args[:])))

		vkFns.CmdDispatch(cb, job.groups[0], job.groups[1], job.groups[2])
	})
}

var errEmptyGrid = errors.New("empty grid")

// TODO: kill
func validateDispatchGrid(validated []uint32, grid []int) error {
	empty := false
	for i, d := range grid {
		if d < 0 {
			return errors.New("bad")
		}
		if d == 0 {
			empty = true
		}
		validated[i] = uint32(d)
	}
	for i := len(grid); i < len(validated); i++ {
		validated[i] = 1
	}
	if empty {
		return errEmptyGrid
	}
	return nil
}

// TODO: limits etc validation
// TODO: rename just to validateGrid
func validateDispatchGrid2(grid []int) error {
	product := 1
	for _, d := range grid {
		if d < 0 {
			return errors.New("bad")
		}
		product *= d
	}
	if product == 0 {
		return errEmptyGrid
	}
	return nil
}
