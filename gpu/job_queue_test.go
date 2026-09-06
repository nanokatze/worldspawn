package gpu

import (
	"testing"
	"time"
	"worldspawn/gpu/vk"
)

type nopJob struct{}

func (*nopJob) Info() JobInfo { return JobInfo{QueueFamilies: ^uint32(0)} }

func (*nopJob) Exec(q *DeviceQueue) {}

func BenchmarkRoundTrip(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		var jq JobQueue
		jq.Enqueue(&nopJob{})
		WaitForIdle(&jq)
	}
}

// Not a real test yet but we're working on it
//
// TODO: make this a benchmark
// TODO: make a variant written in a goroutine per fif way
func TestFramesInFlight(t *testing.T) {
	t.SkipNow()

	A := NewImage(MakeImageConfig(vk.FORMAT_R32_UINT, []int{4096, 4096}).SetLayers(40))
	B := NewImage(MakeImageConfig(vk.FORMAT_R32_UINT, []int{4096, 4096}).SetLayers(40))

	type fif struct {
		wg WaitGroup
	}

	var fifs [2]fif

	last := time.Now()
	for i := range 10 {
		// Wait for the previous commands associated with this frame-in-flight
		// to complete
		current := &fifs[i%len(fifs)]
		current.wg.Wait()

		var jq JobQueue

		// Make sure the new commands happen-after the previous frame's commands
		prev := &fifs[(len(fifs)+i-1)%len(fifs)]
		prev.wg.EnqueueWait(&jq)

		// Very intensive host workload. This should be roughly equal to
		// the time it takes for the device workload to run.
		//
		// TODO: measure this automatically
		time.Sleep(100 * time.Millisecond)

		switch i % 2 {
		case 0:
			EnqueueCopyImage(&jq, A, nil, B, nil, A.Extent())
		case 1:
			EnqueueCopyImage(&jq, B, nil, A, nil, A.Extent())
		}

		current.wg.Add(1)
		current.wg.EnqueueDone(&jq)

		// With len(fifs) > 1, this should be about max(host frame time, device frame time)
		// With len(fifs) == 1, this should be host frametime + device frame time
		now := time.Now()
		t.Log(now.Sub(last))
		last = now
	}

	for i := range fifs {
		fifs[i].wg.Wait()
	}
}

func TestFramesInFlight2(t *testing.T) {
}

func TestJobQueueWaitOnSatisfiedWaitGroup(t *testing.T) {
	var wg WaitGroup

	var jq JobQueue

	wg.EnqueueWait(&jq)

	WaitForIdle(&jq)
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

	WaitForIdle(&jq)
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

	WaitForIdle(&jq)
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

	WaitForIdle(&jq1)
	WaitForIdle(&jq2)
}
