package gpu

import (
	"errors"
	"slices"
	"unsafe"

	"worldspawn/gpu/vk"
)

type dispatchWorkgroupsJob struct {
	groups [3]uint32
	kernel *Func
	args   []byte
}

// TODO: shorter name?
func EnqueueParallelForWorkgroups(jq *JobQueue, groups [3]int, kernel *Func, args any) {
	var validatedGroups [3]uint32
	if err := validateDispatchGrid(groups[:], validatedGroups[:]); err != nil {
		panic(err)
	}
	if slices.Contains(validatedGroups[:], 0) {
		return
	}

	jq.Enqueue(&dispatchWorkgroupsJob{
		groups: validatedGroups,
		kernel: kernel,
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
		vkFns.CmdBindShadersEXT(
			cb,
			1,
			unsafe.SliceData([]vk.ShaderStageFlagBits{vk.SHADER_STAGE_COMPUTE_BIT}),
			unsafe.SliceData([]vk.ShaderEXT{job.kernel.vkShader()}))

		vkFns.CmdBindDescriptorSets(
			cb,
			vk.PIPELINE_BIND_POINT_COMPUTE,
			pipelineLayout,
			0,
			1, &descriptorSet,
			0, nil)

		args := job.args // wtf???
		vkFns.CmdPushConstants(
			cb,
			pipelineLayout,
			vk.ShaderStageFlags(vk.SHADER_STAGE_ALL),
			0,
			uint32(len(args)), unsafe.Pointer(&args))

		vkFns.CmdDispatch(cb, job.groups[0], job.groups[1], job.groups[2])
	})
}

// TODO: swap grid and validated?
func validateDispatchGrid(grid []int, validated []uint32) error {
	if len(grid) != len(validated) {
		return errors.New("horrible")
	}
	for i, d := range grid {
		if d < 0 {
			return errors.New("bad")
		}
		validated[i] = uint32(d)
	}
	return nil
}
