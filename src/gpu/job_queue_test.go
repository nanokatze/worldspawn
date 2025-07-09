package gpu

import (
	"testing"
	"time"
)

type nopJob struct{}

func (*nopJob) Info() JobInfo { return JobInfo{QueueFamilies: ^uint32(0)} }

func (*nopJob) Exec(q *CommandQueue) {}

func BenchmarkRoundTrip(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		var jq JobQueue
		jq.Enqueue(&nopJob{})
		jq.WaitForIdle()
	}
}

func TestFramesInFlight(t *testing.T) {
	type fif struct {
		wg WaitGroup
	}

	var fifs [2]fif

	for i := range 100 {
		var jq JobQueue

		current := &fifs[i%2]
		current.wg.Wait()

		prev := &fifs[(i+1)%2]

		prev.wg.EnqueueWait(&jq)

		// current := &fifs[i%2]

	}
}

func TestJobQueueWaitOnSatisfiedWaitGroup(t *testing.T) {
	var wg WaitGroup

	var jq JobQueue

	wg.EnqueueWait(&jq)

	jq.WaitForIdle()
}

func TestJobQueueWaitOnWaitGroupThatWillBeSatisfiedByHost(t *testing.T) {
	var wg WaitGroup
	wg.Add(1)

	var jq JobQueue

	wg.EnqueueWait(&jq)

	go func() {
		time.Sleep(100 * time.Millisecond)
		wg.Done()
	}()

	jq.WaitForIdle()
}

func TestJobQueueWaitTwiceOnWaitGroupThatWillBeSatisfiedByHost(t *testing.T) {
	var wg WaitGroup
	wg.Add(1)

	var jq JobQueue

	wg.EnqueueWait(&jq)
	wg.EnqueueWait(&jq)

	go func() {
		time.Sleep(100 * time.Millisecond)
		wg.Done()
	}()

	jq.WaitForIdle()
}

// TODO: a more elaborate test
func TestJobQueueChained(t *testing.T) {
	var jq0 JobQueue
	var jq1, jq2 JobQueue

	var linky WaitGroup
	linky.Add(1)
	linky.EnqueueWait(&jq1)
	linky.EnqueueWait(&jq2)
	linky.EnqueueDone(&jq0)

	jq1.WaitForIdle()
	jq2.WaitForIdle()
}
