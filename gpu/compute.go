package gpu

import (
	"errors"
	"runtime"
	"slices"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: rename to something else
type ComputeFuncCode[T any] struct {
	_  [0]T
	vk vk.ShaderEXT
}

// Convenience thing; TODO: probs remove it
func (code ComputeFuncCode[T]) WithData(data T) ComputeFunc[T] {
	return ComputeFunc[T]{
		Code: code,
		Data: data,
	}
}

// TODO: give this a longer name so that we can call an interface ComputeFunc
type ComputeFunc[T any] struct {
	Code ComputeFuncCode[T]
	Data T
}

// TODO: hide
func (f ComputeFunc[T]) ParallelFor(grid []int) Job {
	var grid3 [3]int
	copy(grid3[:], grid)

	return &dispatchJob{
		grid:   grid3,
		kernel: f.Code.vk,
		args:   slices.Clone(asbytes(&f.Data)),
	}
}

/*
type parallelable interface {
	parallelFor(grid [3]int) Job
}

func ParallelFor(grid []int, f parallelable) Job {
	var grid3 [3]int
	copy(grid3[:], grid)
	return f.parallelFor(grid3)
}
*/

// TODO: kill off in favor of ParallelFor
func (f ComputeFunc[T]) Dispatch(jq *JobQueue, grid []int) {
	grid3 := [3]int{1, 1, 1}
	copy(grid3[:], grid)

	jq.Enqueue(&dispatchJob{
		grid:   grid3,
		kernel: f.Code.vk,
		args:   slices.Clone(asbytes(&f.Data)),
	})
}

/*
type lazyComputeFunc[T any] struct {
	_        [0]T
	compiled vk.ShaderEXT
	code     string // TODO: we could also replace this with a function to get []byte probs
}

func (f lazyComputeFunc[T]) Get() vk.ShaderEXT {
	atomic.Load
	return f.compiled
}
*/

// TODO: should be LazyComputeFunc
func CompileFunc[T any](blob []byte, entry string) ComputeFuncCode[T] {
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
	return ComputeFuncCode[T]{
		vk: vkShader,
	}
}

type dispatchJob struct {
	grid   [3]int
	kernel vk.ShaderEXT // TODO: take our own object so that we can debug things etc?
	args   []byte
}

func ParallelFor[T any](jq *JobQueue, grid []int, f ComputeFunc[T]) {
	jq.Enqueue(f.ParallelFor(grid))
}

func (*dispatchJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: queueFamilies.Mask(0b010),
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

		vkFns.CmdDispatch(cb, uint32(job.grid[0]), uint32(job.grid[1]), uint32(job.grid[2]))
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
