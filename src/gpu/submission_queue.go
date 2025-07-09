package gpu

import (
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"worldspawn/gpu/vk"
)

// TODO: merge this file entirely into job_queue.go

// TODO: put this inside schedulerState?
var cqs [32][]*CommandQueue
var cqsflat []*CommandQueue

type CommandQueue struct {
	cb          vk.CommandBuffer
	needBarrier bool

	waits   map[int]uint64
	cmdBufs []vk.CommandBufferSubmitInfo // TODO: replace this with just vk.CommandBuffer and do conversion right at the submit time?
	garbage []func()

	// TODO: rename this to somehing like "in-flight garbage"
	callbacksMu sync.Mutex
	callbacks   []semaphoreSignalCallback // TODO: make this a linked queue

	vkSemaphore vk.Semaphore
	tail        atomic.Uint64 // TODO: is this actually necessary?
	head        uint64

	id int

	queueFamily uint32 // TODO: move this field around idk

	// This is actually the background character here. We could've as well made
	// it be passed on submit.
	vkQueue vk.Queue
}

type semaphoreSignalCallback struct {
	value uint64
	f     func()
}

func newSubmissionQueue(queueFamily, queueIndex uint32) *CommandQueue {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	var vkSemaphore vk.Semaphore
	if err := vkFns.CreateSemaphore(device, &vk.SemaphoreCreateInfo{
		SType: vk.STRUCTURE_TYPE_SEMAPHORE_CREATE_INFO,
		PNext: unsafe.Pointer(pinned(&pinner, &vk.SemaphoreTypeCreateInfo{
			SType:         vk.STRUCTURE_TYPE_SEMAPHORE_TYPE_CREATE_INFO,
			SemaphoreType: vk.SEMAPHORE_TYPE_TIMELINE,
			InitialValue:  0,
		})),
	}, nil, &vkSemaphore); err != nil {
		panic(fmt.Sprintf("gpu: vkCreateSemaphore: %v", err))
	}

	var vkQueue vk.Queue
	vkFns.GetDeviceQueue2(device, &vk.DeviceQueueInfo2{
		SType:            vk.STRUCTURE_TYPE_DEVICE_QUEUE_INFO_2,
		QueueFamilyIndex: queueFamily,
		QueueIndex:       queueIndex,
	}, &vkQueue)

	q := &CommandQueue{
		vkSemaphore: vkSemaphore,
		queueFamily: queueFamily,
		waits:       make(map[int]uint64),
		id:          len(cqsflat),
		vkQueue:     vkQueue,
	}

	cqs[queueFamily] = append(cqs[queueFamily], q)
	cqsflat = append(cqsflat, q)

	go func() {
		ctr := uint64(0)
		for {
			waitSemaphore(q.vkSemaphore, ctr)
			q.tail.Store(ctr)
			for {
				q.callbacksMu.Lock()
				var cb func()
				if len(q.callbacks) > 0 && q.callbacks[0].value <= ctr {
					cb = q.callbacks[0].f
					q.callbacks = q.callbacks[1:]
				}
				q.callbacksMu.Unlock()
				if cb == nil {
					break
				}
				cb()
			}
			ctr++
		}
	}()

	return q
}

// TODO: rename
func (q *CommandQueue) flushbarrier() {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	if q.needBarrier {
		vkFns.CmdPipelineBarrier2(q.cb, &vk.DependencyInfo{
			SType:              vk.STRUCTURE_TYPE_DEPENDENCY_INFO,
			MemoryBarrierCount: 1,
			PMemoryBarriers: pinned(&pinner, &vk.MemoryBarrier2{
				SType:         vk.STRUCTURE_TYPE_MEMORY_BARRIER_2,
				SrcStageMask:  vk.PipelineStageFlags2(vk.PIPELINE_STAGE_2_ALL_COMMANDS_BIT),
				SrcAccessMask: vk.AccessFlags2(vk.ACCESS_2_MEMORY_WRITE_BIT),
				DstStageMask:  vk.PipelineStageFlags2(vk.PIPELINE_STAGE_2_ALL_COMMANDS_BIT),
				DstAccessMask: vk.AccessFlags2(vk.ACCESS_2_MEMORY_READ_BIT) | vk.AccessFlags2(vk.ACCESS_2_MEMORY_WRITE_BIT),
			}),
		})
		q.needBarrier = false
	}
}

func (q *CommandQueue) Commands(f func(cb vk.CommandBuffer)) {
	if q.cb == vk.NULL_HANDLE {
		q.ensurecb()
	}
	f(q.cb)
	q.needBarrier = true
}

// TODO: rename
func (q *CommandQueue) ensurecb() {
	queueFamily := q.queueFamily

	// TODO: outline this
	cb := cbcaches[queueFamily].Get()
	if err := vkFns.BeginCommandBuffer(cb.Vk(), &vk.CommandBufferBeginInfo{
		SType: vk.STRUCTURE_TYPE_COMMAND_BUFFER_BEGIN_INFO,
		Flags: vk.CommandBufferUsageFlags(vk.COMMAND_BUFFER_USAGE_ONE_TIME_SUBMIT_BIT),
	}); err != nil {
		panic(fmt.Sprintf("gpu: vkBeginCommandBuffer: %v", err))
	}

	q.cb = cb.Vk()
	q.Cleanup(func() { cbcaches[queueFamily].Put(cb) })
}

// cb must be encoded for the queue family that q submits to
func (q *CommandQueue) CommandBuffer(cb vk.CommandBuffer) {
	q.flushcb()
	q.cb = cb
	q.needBarrier = true
}

func (q *CommandQueue) QueueOperation(f func(vkQueue vk.Queue)) {
	q.submit()
	f(q.vkQueue)
}

func (q *CommandQueue) flushcb() {
	if q.cb != vk.NULL_HANDLE {
		q.actuallyflushcb()
	}
}

func (q *CommandQueue) actuallyflushcb() {
	// TODO: outline this
	// TODO: defer closing cmdbufs until right before submission? That would
	// let us prepend barriers nicely
	if err := vkFns.EndCommandBuffer(q.cb); err != nil {
		panic(fmt.Sprintf("gpu: vkEndCommandBuffer: %v", err))
	}
	q.cmdBufs = append(q.cmdBufs,
		vk.CommandBufferSubmitInfo{
			SType:         vk.STRUCTURE_TYPE_COMMAND_BUFFER_SUBMIT_INFO,
			CommandBuffer: q.cb,
		})
	q.cb = vk.NULL_HANDLE
}

func (q *CommandQueue) Cleanup(f func()) {
	q.garbage = append(q.garbage, f)
}

func (q *CommandQueue) String() string {
	// TODO: a nicer string
	return fmt.Sprintf("command queue id=%d family=%d", q.id, q.queueFamily)
}

// TODO: rename
func (q *CommandQueue) submit() {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	q.flushbarrier()
	q.flushcb()

	// TODO: outline the stuff we flush here into its own struct

	// TODO: we can handle garbage without submitting

	if len(q.cmdBufs) == 0 && len(q.garbage) == 0 {
		return
	}

	var waits []vk.SemaphoreSubmitInfo
	for id, value := range q.waits {
		// TODO: make the counter in q accessible externally so we can know when
		// it has already been signalled so we don't need to make the cmdbufs
		// wait on it.

		p := cqsflat[id]

		// if p.tail.Load()

		waits = append(waits,
			vk.SemaphoreSubmitInfo{
				SType:     vk.STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
				Semaphore: p.vkSemaphore,
				Value:     value,
				StageMask: vk.PipelineStageFlags2(vk.PIPELINE_STAGE_2_ALL_COMMANDS_BIT),
			})
	}

	cmdBufs := q.cmdBufs
	q.cmdBufs = nil

	garbage := q.garbage
	q.garbage = nil

	// Append the callback before submitting so that we don't miss the signal.
	q.callbacksMu.Lock()
	q.callbacks = append(q.callbacks,
		semaphoreSignalCallback{
			value: q.head + 1,
			f: func() {
				for _, g := range garbage {
					g()
				}
			},
		})
	q.callbacksMu.Unlock()

	// TODO: submit is a bit expensive (on the order of 20 microseconds), we
	// might want to consider running a separate goroutine for submit calls. Or
	// for any calls on CommandQueue at all actually.
	if err := vkFns.QueueSubmit2(q.vkQueue, 1, &vk.SubmitInfo2{
		SType:                    vk.STRUCTURE_TYPE_SUBMIT_INFO_2,
		WaitSemaphoreInfoCount:   uint32(len(waits)),
		PWaitSemaphoreInfos:      pinnedSliceData(&pinner, waits),
		CommandBufferInfoCount:   uint32(len(cmdBufs)),
		PCommandBufferInfos:      pinnedSliceData(&pinner, cmdBufs),
		SignalSemaphoreInfoCount: 1,
		PSignalSemaphoreInfos: pinned(&pinner, &vk.SemaphoreSubmitInfo{
			SType:     vk.STRUCTURE_TYPE_SEMAPHORE_SUBMIT_INFO,
			Semaphore: q.vkSemaphore,
			Value:     q.head + 1,
			StageMask: vk.PipelineStageFlags2(vk.PIPELINE_STAGE_2_ALL_COMMANDS_BIT),
		}),
	}, vk.NULL_HANDLE); err != nil {
		panic(fmt.Sprintf("gpu: vkQueueSubmit2: %v", err))
	}
	q.head++
	q.waits[q.id] = q.head
}

func waitSemaphore(semaphore vk.Semaphore, value uint64) {
	var pinner runtime.Pinner
	defer pinner.Unpin()

	if err := vkFns.WaitSemaphores(device, &vk.SemaphoreWaitInfo{
		SType:          vk.STRUCTURE_TYPE_SEMAPHORE_WAIT_INFO,
		SemaphoreCount: 1,
		PSemaphores:    pinned(&pinner, &semaphore),
		PValues:        pinned(&pinner, &value),
	}, math.MaxUint64); err != nil {
		panic(fmt.Sprintf("gpu: vkWaitSemaphores: %v", err))
	}
}
