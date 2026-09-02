package ktx2

import (
	"io"
	"slices"

	"worldspawn/gpu"
)

// Like slices.Index, but returns len(s) instead of -1.
func index[S ~[]E, E comparable](s S, v E) int {
	if i := slices.Index(s, v); i >= 0 {
		return i
	}
	return len(s)
}

// TODO: move this into gpu?
func enqueueReadAt(jq *gpu.JobQueue, r io.ReaderAt, p gpu.Slice[byte], off int64) {
	enqueueHostCall(jq, func() {
		if _, err := r.ReadAt(p.Value(), off); err != nil {
			// We don't really have any way to report read failures for now.
			panic(err)
		}
	})
}

func enqueueWriteAt(jq *gpu.JobQueue, w io.WriterAt, p gpu.Slice[byte], off int64) {
	enqueueHostCall(jq, func() {
		if _, err := w.WriteAt(p.Value(), off); err != nil {
			// We don't really have any way to report read failures for now.
			panic(err)
		}
	})
}

// TODO: move into gpu?
// TODO: come up with a better name?
func enqueueHostCall(jq *gpu.JobQueue, f func()) {
	var wg1 gpu.WaitGroup
	wg1.Add(1)
	var wg2 gpu.WaitGroup
	wg2.Add(1)

	wg1.EnqueueDone(jq)
	go func() {
		wg1.Wait()
		f()
		wg2.Done()
	}()
	wg2.EnqueueWait(jq)
}
