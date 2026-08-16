package gpu

import (
	"errors"
	"runtime"
	"slices"
	"unsafe"

	"worldspawn/gpu/vk"
)

type ComputeShader[T any] struct {
	_  [0]T
	vk vk.ShaderEXT
}

type ComputeClosure[T any] struct {
	shader *ComputeShader[T]
	env    T
}

// TODO: also specify type of the blob. We could encode the type into Go's
// typesystem (i.e. just newtype []byte.)
func CompileComputeShader[T any](blob []byte, entry string) *ComputeShader[T] {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	gpuInit()

	var vkShader vk.ShaderEXT
	must(vkFns.CreateShadersEXT(device, 1, &vk.ShaderCreateInfoEXT{
		SType:    vk.STRUCTURE_TYPE_SHADER_CREATE_INFO_EXT,
		Flags:    vk.ShaderCreateFlagsEXT(vk.SHADER_CREATE_DESCRIPTOR_HEAP_BIT_EXT),
		Stage:    vk.SHADER_STAGE_COMPUTE_BIT,
		CodeType: vk.SHADER_CODE_TYPE_SPIRV_EXT,
		CodeSize: len(blob),
		PCode:    unsafe.Pointer(pinnedSliceData(&pinner, blob)),
		PName:    pinnedCString(&pinner, entry),
	}, nil, &vkShader))

	// TODO: stick a cleanup func on the resulting object once we properly
	// maintain the reference and stuff?
	return &ComputeShader[T]{vk: vkShader}
}

func (shader *ComputeShader[T]) Bind(env T) ComputeClosure[T] {
	return ComputeClosure[T]{shader, env}
}

type dispatchJob struct {
	groups [3]uint32
	kernel vk.ShaderEXT // TODO: make it be generic over compute compute shader objects so that we can keep the reference
	args   []byte
}

// TODO: play with the order of arguments?
// TODO: should take an interface rather than be generic. Or maybe not.
func ParallelFor[T any](jq *JobQueue, groups []int, f ComputeClosure[T]) {
	if err := validateGrid(groups); err != nil {
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
		kernel: f.shader.vk,
		args:   slices.Clone(asbytes(&f.env)),
	})
}

func (*dispatchJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: topology.QueueFamilies(vk.QueueFlags(vk.QUEUE_COMPUTE_BIT)),
	}
}

func (job *dispatchJob) Exec(q *DeviceQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		var pinner runtime.Pinner
		defer pinner.Unpin()

		BindDescriptorHeaps(cb)

		vkFns.CmdBindShadersEXT(
			cb,
			1,
			unsafe.SliceData([]vk.ShaderStageFlagBits{vk.SHADER_STAGE_COMPUTE_BIT}),
			unsafe.SliceData([]vk.ShaderEXT{job.kernel}))

		pinner.Pin(unsafe.SliceData(job.args))

		vkFns.CmdPushDataEXT(cb, &vk.PushDataInfoEXT{
			SType: vk.STRUCTURE_TYPE_PUSH_DATA_INFO_EXT,
			Data: vk.HostAddressRangeConstEXT{
				Address: unsafe.Pointer(unsafe.SliceData(job.args)),
				Size:    len(job.args),
			},
		})

		vkFns.CmdDispatch(cb, job.groups[0], job.groups[1], job.groups[2])
	})
}

var errEmptyGrid = errors.New("empty grid")

// TODO: limits etc validation
// TODO: rename just to validateGrid
func validateGrid(grid []int) error {
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
