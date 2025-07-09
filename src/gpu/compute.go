package gpu

import (
	"slices"
	"unsafe"

	"worldspawn/gpu/vk"
)

type dispatchJob struct {
	n      uint32
	kernel *Func
	args   []byte
}

func EnqueueParallelFor(jq *JobQueue, n int, kernel *Func, args any) {
	if n < 0 {
		panic("bad")
	}
	if n == 0 {
		return
	}

	// TODO: validate that n fits an uint32

	jq.Enqueue(&dispatchJob{
		n:      uint32(n),
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

		vkFns.CmdDispatch(cb, job.n, 1, 1)
	})
}
