package gpu

import (
	"slices"
	"unsafe"

	"worldspawn/gpu/vk"
)

type dispatchJob struct {
	groups [3]uint32
	kernel *Func
	args   []byte
}

func EnqueueParallelForGroups(jq *JobQueue, groups [3]int, kernel *Func, args any) {
	validatedGroups := validateDispatchDimensions(groups)

	jq.Enqueue(&dispatchJob{
		groups: validatedGroups,
		kernel: kernel,
		args:   slices.Clone(asbytes(args)),
	})
}

func (*dispatchJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: queueFamilies.Mask(0b010),
	}
}

func (job *dispatchJob) Exec(q *CommandQueue) {
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

func validateDispatchDimensions(x [3]int) [3]uint32 {
	if x[0] < 0 || x[1] < 0 || x[2] < 0 {
		panic("bad")
	}
	return [3]uint32{uint32(x[0]), uint32(x[1]), uint32(x[2])}
}
