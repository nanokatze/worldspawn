package gpu

import (
	"errors"
	"slices"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: use "threadgroups" instead of "workgroups"? Or just groups.

type dispatchWorkgroupsJob struct {
	groups [3]uint32
	kernel *Func
	args   []byte
}

// TODO: shorter name?
func ParallelForWorkgroups(jq *JobQueue, groups []int, f *Func, args any) {
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

	jq.Enqueue(&dispatchWorkgroupsJob{
		groups: groups32,
		kernel: f,
		args:   slices.Clone(asbytes(args)),
	})
}

func (*dispatchWorkgroupsJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: queueFamilies.Mask(0b010),
	}
}

func (job *dispatchWorkgroupsJob) Exec(q *CommandQueue) {
	q.Commands(func(cb vk.CommandBuffer) {
		BindDescriptorSet(cb, vk.PIPELINE_BIND_POINT_COMPUTE)

		vkFns.CmdBindShadersEXT(
			cb,
			1,
			unsafe.SliceData([]vk.ShaderStageFlagBits{vk.SHADER_STAGE_COMPUTE_BIT}),
			unsafe.SliceData([]vk.ShaderEXT{job.kernel.vkShader()}))

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
