package gpu

import (
	"log"
	"testing"
	"time"
	"worldspawn/gpu/vk"
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

	A := NewImage(&ImageConfig{
		Dim:       ImageDim2D,
		Extent:    [3]int{4096, 4096, 1},
		Layers:    40,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_R32_UINT,
	})
	B := NewImage(&ImageConfig{
		Dim:       ImageDim2D,
		Extent:    [3]int{4096, 4096, 1},
		Layers:    40,
		MipLevels: 1,
		Samples:   1,
		Format:    vk.FORMAT_R32_UINT,
	})

	var fifs [2]fif

	last := time.Now()

	for i := range 100 {
		var jq JobQueue

		current := &fifs[i%len(fifs)]
		current.wg.Wait()

		time.Sleep(100 * time.Millisecond)

		prev := &fifs[(len(fifs)+i-1)%len(fifs)]
		prev.wg.EnqueueWait(&jq)

		switch i % 2 {
		case 0:
			EnqueueCopyImage(&jq, A, [3]int{}, B, [3]int{}, A.Extent())
		case 1:
			EnqueueCopyImage(&jq, B, [3]int{}, A, [3]int{}, A.Extent())
		}

		// record stuff into jq here

		current.wg.Add(1)
		current.wg.EnqueueDone(&jq)

		now := time.Now()
		log.Println(now.Sub(last))
		last = now
	}

	for i := range fifs {
		fifs[i].wg.Wait()
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
