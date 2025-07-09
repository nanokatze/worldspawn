package gpu

import (
	"fmt"
	"math/rand/v2"
	"sync/atomic"

	"worldspawn/gpu/internal/bitset"
)

// TODO: move the entire thing in internal/slot_alloc or something?
type slotAlloc struct{ bs bitset.Bitset }

func newSlotAlloc(len int) slotAlloc {
	bs := bitset.Make(len)
	bs.Set(0)

	return slotAlloc{bs: bs}
}

// TODO: can we somehow make hint per-g/per-m?
func (a slotAlloc) Alloc(hint *int64) int {
	h := atomic.LoadInt64(hint)

	i0 := int(h)
	if i0 == 0 {
		// Hint is uninitialized. Choose the start index at random in hopes that
		// we won't contend with others.
		i0 = rand.IntN(a.bs.Len())
	}
	// Round down the start index to the bitset's word boundary.
	i0 = i0 / 64 * 64

	i := a.bs.FindAndSet(i0)
	if i < 0 {
		// Try again, starting at 0.
		i = a.bs.FindAndSet(0)
	}

	if i == 0 {
		// Slot 0 is reserved
		panic("unreachable")
	}
	if i < 0 {
		panic("out of free slots")
	}

	if h != int64(i) {
		// We picked an index different from the hint. Try to update the hint,
		// but don't bother if someone already has done so, to avoid cache line
		// ping-pong.
		atomic.CompareAndSwapInt64(hint, h, int64(i))
	}

	return i
}

func (a *slotAlloc) Free(i int) {
	if i == 0 {
		return
	}

	if !a.bs.Unset(i) {
		panic(fmt.Sprintf("tried to free slot %d that was not allocated", i))
	}
}
