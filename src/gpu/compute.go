package gpu

import (
	"slices"
	"unsafe"

	"worldspawn/gpu/vk"
)

type dispatchJob struct {
	x      uint32
	y      uint32
	z      uint32
	kernel *Func
	args   []byte
}

// TODO: replace n with a range like in tbb/sycl?
func EnqueueParallelFor(jq *JobQueue, n int, kernel *Func, args any) {
	if n < 0 {
		panic("bad")
	}
	if n == 0 {
		return
	}

	// TODO: validate that n fits an uint32

	jq.Enqueue(&dispatchJob{
		x:      uint32(n),
		y:      1,
		z:      1,
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

		vkFns.CmdDispatch(cb, job.x, job.y, job.z)
	})
}
