package gpu

import (
	"sync"
	"sync/atomic"
)

// TODO: make a lighter weight sync primitive with strong assumptions, for
// internal use by us (scheduler, splitting job queues, etc.) We need it to be
// at most one shot (signaled exactly once to become satisfied) and support host
// wait (but host signal is not necessary.) Its API should also be in terms of
// batch rather than job queue so we don't need it to handle splitting. In fact
// we'll use it to implement splitting.

/*
type oneshot struct {
	mu sync.Mutex

	signaled               bool
	signaledSpeculatively  bool
	speculativeSignalQueue int
	speculativeSignalValue uint64

	notifyList
}
*/

// TODO: let's not expose any bits for implementing sync primitives externally.
// Sync primitives should be built on top of stuff we provide here.

// TODO: the implementation is rather messy, we should clean it up

// TODO: document this better, incl. internals

type WaitGroup struct {
	ctr atomic.Uint32

	mu sync.Mutex

	notify notifyList
}

func (wg *WaitGroup) init() {
	wg.notify.init(&wg.mu)
}

// Add must not be called once there are any waiters, either host goroutines or
// JobQueues.
func (wg *WaitGroup) Add(delta int) {
	if delta < 0 {
		panic("negative delta")
	}
	if delta == 0 {
		return
	}

	if wg.ctr.Add(uint32(delta)) == uint32(delta) {
		wg.mu.Lock()
		defer wg.mu.Unlock()

		wg.init()
	}
}

func (wg *WaitGroup) Done() {
	ctr := wg.ctr.Add(^uint32(0))
	if ctr == ^uint32(0) {
		panic("negative counter")
	}
	if ctr == 0 {
		wg.mu.Lock()
		defer wg.mu.Unlock()

		wg.notify.notify()
	}
}

func (wg *WaitGroup) Wait() {
	if wg.ctr.Load() == 0 {
		// Already satisfied
		return
	}

	wg.mu.Lock()
	defer wg.mu.Unlock()

	// TODO: write an explanation for why we wake the scheduler urgently
	// here
	ScheduleNow()

	for wg.ctr.Load() != 0 {
		wg.notify.goroutines.Wait()
	}
}

// TODO: rename this to EnqueueDoneInto?
func (wg *WaitGroup) EnqueueDone(jq *JobQueue) {
	// TODO: use host Done() when possible

	b := jq.b()

	b.lock()
	defer b.unlockAndMaybeMakeRunnable()

	b.enqueueSignalLocked(wg)
}

// TODO: rename this to EnqueueWaitInto?
func (wg *WaitGroup) EnqueueWait(jq *JobQueue) {
	if wg.ctr.Load() == 0 {
		return
	}

	wg.mu.Lock()
	defer wg.mu.Unlock()

	// TODO: explain better why we need to check the counter the second time
	// here
	//
	// Assume threads T1 performing EnqueueWait, T2 performing Done
	// T1: we check wg.ctr.Load() and it's 1
	// T2: decrements the counter
	// T2: locks the wg.mu and notifies everyone on the notifyList
	// T1: we put jq on the notify list that was already notified
	// Doing this check prevents this scenario.
	if wg.ctr.Load() > 0 {
		jq.prepareForEnqueueWait()
		wg.enqueueWaitIntoBatch(jq.b())
	}
}

/*
// TODO: uncomment this when we have actual users for it
// TODO: rename
// TODO: equivalent host method called Go
func (wg *WaitGroup) Helper(f func(jq *JobQueue)) {
	wg.Add(1)

	var jq JobQueue
	f(&jq)
	wg.EnqueueDone(&jq)
}
*/

// Must be called with wg.mu held
// TODO: get rid of this
func (wg *WaitGroup) enqueueWaitIntoBatch(b *jobBatch) {
	b.lock()
	defer b.unlock()

	wg.enqueueWaitIntoBatchLocked(b)
}

// Must be called with wg.mu held
func (wg *WaitGroup) enqueueWaitIntoBatchLocked(b *jobBatch) {
	// TODO: rewrite these messages to be more concise
	if !b.empty() {
		panic("trying to enqueue wait into a batch that has jobs")
	}
	if b.runnable {
		panic("trying to enqueue wait into a batch that's in the runnable set")
	}

	b.deps++

	wg.notify.addJobBatch(b)
}
