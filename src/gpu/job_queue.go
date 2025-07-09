package gpu

import (
	"math/rand/v2"
	"sync"
	"time"
)

// TODO: rename to something more specific?
type JobInfo struct {
	// QueueFamilies is the set of queue families this job can be executed on
	QueueFamilies uint32
}

type Job interface {
	Info() JobInfo

	// TODO: rename this to something better. E.g. EmitCommands or
	// RecordCommands or just Emit?
	Exec(q *CommandQueue)
}

type JobQueue struct {
	currentBatch *jobBatch // use b() to access
	donewg       *WaitGroup
	// TODO: add some sort of mechanism so that things can't be enqueued while
	// we have a render pass using this job queue
}

func (jq *JobQueue) b() *jobBatch {
	if jq.currentBatch == nil {
		jq.currentBatch = new(jobBatch)
	}
	// We might be about to have a job enqueued. Reset done so a new sync object
	// ordered to be signaled after the new jobs complete is created if
	// necessary.
	jq.donewg = nil
	return jq.currentBatch
}

// prepareForEnqueueWait must be called before enqueueing a wait into the job
// queue.
func (jq *JobQueue) prepareForEnqueueWait() {
	if jq.currentBatch == nil || jq.currentBatch.empty() {
		return
	}
	done := jq.done()
	jq.currentBatch = nil
	done.enqueueWaitIntoBatch(jq.b())
}

func (jq *JobQueue) Enqueue(job Job) {
	jq.b().enqueue(job)
}

// TODO: rename to EnqueueCleanup?
func (jq *JobQueue) Cleanup(f func()) {
	jq.b().enqueueCleanup(f)
}

func (jq *JobQueue) done() *WaitGroup {
	if jq.donewg == nil {
		wg := new(WaitGroup)
		wg.Add(1)
		wg.EnqueueDone(jq)
		jq.donewg = wg
	}
	return jq.donewg
}

func (jq *JobQueue) Fork() *JobQueue {
	child := new(JobQueue)
	jq.done().EnqueueWait(child)
	return child
}

func (jq *JobQueue) WaitForIdle() {
	jq.done().Wait()
}

// TODO: region api for debug tooling and stuff

type jobBatch struct {
	mu sync.Mutex

	// deps == 0 ^ !runnable
	// runnable iff runnable contains this batch

	deps    int // number of unsatisfied dependencies
	jobs    []Job
	tailJob *tailJob

	runnable bool // whether this batch is in the runnable set
}

var runnable sync.Map // map[*jobBatch]struct{}

func (b *jobBatch) dependenciesSatisfied() bool {
	return b.deps == 0
}

func (b *jobBatch) empty() bool {
	return len(b.jobs) == 0 && b.tailJob == nil
}

func (b *jobBatch) enqueue(job Job) {
	// TODO: performance

	b.lock()
	defer b.unlockAndMaybeMakeRunnable()

	b.flushTail()
	b.jobs = append(b.jobs, job)
}

type tailJob struct {
	// TODO: explain why we use slice instead of map
	// TODO: make signals non-WaitGroup specific
	signals []*WaitGroup

	garbage []func()

	// TODO: see if we would benefit from small storage in tailJob
}

func (*tailJob) Info() JobInfo {
	return JobInfo{
		QueueFamilies: ^uint32(0),
	}
}

func (*tailJob) Exec(*CommandQueue) { panic("unreachable") }

// Must be called with b locked.
func (b *jobBatch) tail() *tailJob {
	if b.tailJob == nil {
		b.tailJob = new(tailJob)
	}
	return b.tailJob
}

// Must be called with b locked.
// TODO: make it non-WaitGroup-specific
func (b *jobBatch) enqueueSignalLocked(wg *WaitGroup) {
	tail := b.tail()
	tail.signals = append(tail.signals, wg)
}

func (b *jobBatch) enqueueCleanup(f func()) {
	b.lock()
	defer b.unlockAndMaybeMakeRunnable()

	tail := b.tail()
	tail.garbage = append(tail.garbage, f)
}

func (b *jobBatch) flushTail() {
	if b.tailJob != nil {
		b.jobs = append(b.jobs, b.tailJob)
		b.tailJob = nil
	}
}

func (b *jobBatch) lock() { b.mu.Lock() }

func (b *jobBatch) unlock() { b.mu.Unlock() }

func (b *jobBatch) unlockAndMaybeMakeRunnable() {
	defer b.mu.Unlock()

	if !b.dependenciesSatisfied() {
		return
	}

	// Avoid burdening the scheduler with empty batches.
	//
	// We may end up here if all dependencies of a batch become satisfied before
	// any jobs are enqueued.
	if b.empty() {
		return
	}

	if !b.runnable {
		runnable.LoadOrStore(b, struct{}{})
		b.runnable = true

		// TODO: call this without b.mu held
		schedule()
	}
}

// TODO: fill this out
type wakeupReason struct {
	time time.Time
	// callers [1]uintptr // TODO
}

// TODO: rename
func makeWakeupReason(skip int) wakeupReason {
	// var pcs [1]uintptr
	// runtime.Callers(skip+1, pcs[:])
	// return wakeupReason{pcs: pcs}
	return wakeupReason{time: time.Now()}
}

// 1-buffered so that wakeups aren't missed
var (
	scheduleCh    = make(chan wakeupReason, 1)
	scheduleNowCh = make(chan wakeupReason, 1)
)

func schedule() {
	select {
	case scheduleCh <- makeWakeupReason(2):
		schedInit()
	default:
	}
}

func ScheduleNow() {
	select {
	case scheduleNowCh <- makeWakeupReason(2):
		schedInit()
	default:
	}
}

var schedInitOnce sync.Once

func schedInit() {
	schedInitOnce.Do(func() {
		gpuInit()

		go func() {
			for {
				reason := <-scheduleCh

				// TODO: make the delay configurable
				d := 5 * time.Millisecond
				time.Sleep(d)

				// TODO: append delay information to reason

				select {
				case scheduleNowCh <- reason:
				default:
				}
			}
		}()

		go func() {
			scheduler := schedState{
				queues: make(map[*CommandQueue]map[*schedBatch]struct{}),
			}
			for {
				<-scheduleNowCh
				scheduler.schedule()
			}
		}()
	})
}

// TODO: group jobs together so we can poke a single Exec for all of them

type schedState struct {
	queues map[*CommandQueue]map[*schedBatch]struct{}
}

type schedBatch struct {
	jobs []Job
	next int
}

// TODO: print the final schedule to somewhere for debugging
// TODO: if we ever support suspending in the middle of scheduling, the returned
// bool should indicate if we did so or did we finished scheduling everything.
func (sched *schedState) schedule() bool {
	runnable.Range(func(k, _ any) bool {
		b := k.(*jobBatch)

		b.lock()

		if !b.runnable {
			panic("unreachable")
		}
		if !b.dependenciesSatisfied() {
			panic("runnable set contains a batch with unsatisfied dependencies")
		}
		if b.empty() {
			panic("runnable set contains a batch with no jobs")
		}

		var wg WaitGroup
		wg.Add(1)

		b.enqueueSignalLocked(&wg)

		b.flushTail()
		jobs := b.jobs
		b.jobs = nil

		// Remove this batch from the runnable set.
		runnable.Delete(b)
		b.runnable = false

		// Make the future jobs enqueued into this batch wait for the work we're
		// about to schedule to complete.
		wg.mu.Lock()
		wg.enqueueWaitIntoBatchLocked(b)
		wg.mu.Unlock()

		b.unlock()

		b2 := &schedBatch{jobs: jobs}

		q := sched.chooseQueueForBatch(b2, nil)
		if sched.queues[q] == nil {
			sched.queues[q] = make(map[*schedBatch]struct{})
		}
		sched.queues[q][b2] = struct{}{}

		return true
	})

	// TODO: submits and, to lesser extent, recording vulkan commands, do take a
	// bit of cpu time. See how we could distribute work over multiple
	// goroutines so our wall clock time is less.

	for len(sched.queues) > 0 {
		// TODO: clean up migration-related code

		type migration struct {
			to, from *CommandQueue
			batch    *schedBatch
		}

		// TODO: if desired, this slice could be split up by target
		// *CommandQueues to distribute locking
		var migrations []migration
		// TODO: this loop is parallelizable but it runs on the order of 100
		// microseconds
		for q, batches := range sched.queues {
			// TODO: sort/group jobs so we don't e.g. flush CommandQueue in the
			// middle of recording commands because a random job poked
			// QueueOperation

			for b := range batches {
				switch job := b.jobs[b.next].(type) {
				case *tailJob:
					// handled below

				default:
					job.Exec(q)
					b.next++
				}

				// Schedule the tail jobs

				for ; b.next < len(b.jobs); b.next++ {
					tail, ok := b.jobs[b.next].(*tailJob)
					if !ok {
						break
					}

					// TODO: reintroduce speculative signals

					q.Cleanup(func() {
						// TODO: have special handling for signals in CommandQueue
						// itself?
						for _, s := range tail.signals {
							s.Done()
						}

						// TODO: would waking up the scheduler urgently after signaling
						// the signal improve round trip latency?

						for _, g := range tail.garbage {
							g()
						}
					})
				}

				if b.next < len(b.jobs) {
					q2 := sched.chooseQueueForBatch(b, q)
					if q != q2 {
						delete(batches, b)

						migrations = append(migrations,
							migration{
								to:    q2,
								from:  q,
								batch: b,
							})
					}
				} else {
					delete(batches, b)
				}
			}

			q.flushbarrier()
		}

		// TODO: instead of another pass over all sched.queues, we could
		// introduce a deletion bitmap and set the bit in the body of the loop
		// over queues.
		for q, batches := range sched.queues {
			if len(batches) == 0 {
				q.submit()

				delete(sched.queues, q)
			}
		}

		for _, m := range migrations {
			m.from.submit()
			m.to.submit()
		}
		for _, m := range migrations {
			m.to.waits[m.from.id] = max(m.to.waits[m.from.id], m.from.head)

			if sched.queues[m.to] == nil {
				sched.queues[m.to] = make(map[*schedBatch]struct{})
			}
			sched.queues[m.to][m.batch] = struct{}{}
		}
	}

	return true
}

func (sched *schedState) chooseQueueForBatch(b *schedBatch, current *CommandQueue) *CommandQueue {
	families := queueFamilies.Mask(0)

	families &= b.jobs[b.next].Info().QueueFamilies

	if true {
		// If the current queue is ok, stay on it
		if current != nil && families&(1<<current.queueFamily) != 0 {
			return current
		}

		// Look at the jobs ahead so we don't need to migrate
		for _, job := range b.jobs[b.next:] {
			families &= job.Info().QueueFamilies
		}
	}

	// TODO: be smarter here
	for _, family := range queueFamilies.probe {
		if families&(1<<family) != 0 {
			return cqs[family][rand.IntN(len(cqs[family]))]
		}
	}

	// TODO: print scheduler log here
	panic("failed to choose a queue for a batch")
}
