package draw

import (
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

type job struct {
	cb      vk.CommandBuffer
	garbage []func()

	queueFamily int
}

func (job *job) Info() gpu.JobInfo {
	return gpu.JobInfo{
		QueueFamilies: 1 << job.queueFamily,
	}
}

func (job *job) Exec(q *gpu.DeviceQueue) {
	q.CommandBuffer(job.cb)

	q.Cleanup(func() {
		for _, g := range job.garbage {
			g()
		}
	})
}
