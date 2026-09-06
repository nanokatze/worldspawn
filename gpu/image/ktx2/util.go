package ktx2

import (
	"io"
	"slices"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"

	"golang.org/x/exp/constraints"
)

// TODO: this feels like it belongs to either gpu/image or gpu/vk/formatutil
func calcLinearSize(format vk.Format, extent []int, layers int) int {
	formatDesc := formatutil.Describe(format)

	blocks := 1
	for i, side := range extent {
		blocks *= divRoundUp(side, formatDesc.BlockExtent[i])
	}
	return layers * blocks * formatDesc.BlockSize
}

func roundUp[T constraints.Integer](x, multiple T) T {
	// TODO: don't do this stupidity
	for x%multiple != 0 {
		x++
	}
	return x
}

func divRoundUp[T constraints.Integer](x, y T) T { return (x + y - 1) / y }

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
