package slotalloc

import (
	"fmt"
	"math/rand/v2"
	"sync/atomic"

	"worldspawn/gpu/internal/bitslice"
)

type Slotalloc struct{ bs bitslice.BitSlice }

func Make(capacity int) Slotalloc {
	return Slotalloc{bs: bitslice.Make(capacity)}
}

func (a Slotalloc) Cap() int { return a.bs.Len() }

func (a Slotalloc) AllocAt(i int) int {
	if a.bs.Swap(i, true) != false {
		return -1
	}
	return i
}

// TODO: we need to be able to alloc a run of N bits
func (a Slotalloc) Alloc(hintp *int64) int {
	hint := atomic.LoadInt64(hintp)

	i := int(hint)
	if i == 0 {
		// Hint was uninitialized. Choose the start index at random in hopes
		// that we won't contend with others.
		i = rand.IntN(a.bs.Len())
	}
	// Round down the start index to the bitset's word boundary.
	i = i / 64 * 64

	i = a.bs.FindAndSet(i)
	if i < 0 {
		// Try again, starting at 0.
		i = a.bs.FindAndSet(0)
	}

	if i >= 0 {
		if hint != int64(i) {
			// We succeeded and picked an index different from the hint. Try to
			// update the hint, but don't bother if someone already has done so,
			// to avoid cache line ping-pong.
			atomic.CompareAndSwapInt64(hintp, hint, int64(i))
		}
	}
	return i
}

func (a Slotalloc) Free(i int) {
	if a.bs.Swap(i, false) != true {
		panic(fmt.Sprintf("tried to free slot %d that was not allocated", i))
	}
}
