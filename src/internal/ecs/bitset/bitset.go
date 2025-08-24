package bitset

import (
	"fmt"
	"iter"
	"sync/atomic"
)

type boundsError struct {
	x int
	y int
}

func (e boundsError) Error() string {
	return fmt.Sprintf("ecs/hbitset: index out of range [%v] with length %v", e.x, e.y)
}

const (
	wordBits = 64
	ctr0Bits = 32 * wordBits
)

// All operations on Bitset have the same semantics and no stronger
// synchronization requirements than their []bool counterparts.
type Bitset struct {
	words []word
	ctrs0 []uint32

	// TODO: see if we'll benefit from one extra level of counters

	len int
}

func Make(len int) Bitset {
	return Bitset{
		words: make([]word, divRoundUp(len, wordBits)),
		ctrs0: make([]uint32, divRoundUp(len, ctr0Bits)),
		len:   len,
	}
}

func Copy(dst, src Bitset) {
	// TODO: could be made faster on sparse bitsets by checking the counters

	copy(dst.words, src.words)
	copy(dst.ctrs0, src.ctrs0)
}

func (bs Bitset) Test(i int) bool {
	if i < 0 || bs.len <= i {
		panic(boundsError{x: i, y: bs.len})
	}

	// Checking the counters is not worth it

	mask := word(1) << (i % wordBits)

	return atomicLoadWord(&bs.words[i/wordBits])&mask != 0
}

// func (bs BitSet) FindAndSet()

// Set sets the bit i and returns the old value.
func (bs Bitset) Set(i int) bool {
	if i < 0 || i >= bs.len {
		panic(boundsError{x: i, y: bs.len})
	}

	mask := word(1) << (i % wordBits)

	old := atomicOrWord(&bs.words[i/wordBits], mask)
	if old == 0 {
		// The word has become non-zero, increment the counter
		atomic.AddUint32(&bs.ctrs0[i/ctr0Bits], 1)
	}
	return old&mask != 0
}

// Unset unsets the bit i and returns the old value.
func (bs Bitset) Unset(i int) bool {
	if i < 0 || bs.len <= i {
		panic(boundsError{x: i, y: bs.len})
	}

	mask := word(1) << (i % wordBits)

	old := atomicAndWord(&bs.words[i/wordBits], ^mask)
	if old == mask {
		// The word has become zero, decrement the counter
		atomic.AddUint32(&bs.ctrs0[i/ctr0Bits], ^uint32(0))
	}
	return old&mask != 0
}

func (bs Bitset) Reset() {
	// TODO: could be made faster on sparse bitsets by checking the counters

	clear(bs.words)
	clear(bs.ctrs0)
}

// TODO: add a Clear(v bool) method?

// And returns an iterator over elements present in all bss.
func And(bss ...Bitset) iter.Seq[int] {
	return func(yield func(int) bool) {
		if len(bss) == 0 {
			return
		}

		n := len(bss[0].words) * wordBits

	Ctr0Loop:
		for i := 0; i < n; i += ctr0Bits {
			for _, bs := range bss {
				if atomic.LoadUint32(&bs.ctrs0[i/ctr0Bits]) == 0 {
					continue Ctr0Loop
				}
			}

		WordLoop:
			for j := i; j < min(i+ctr0Bits, n); j += wordBits {
				// TODO: outline this into wordIter when doing so stops
				// regressing the benchmarks. Outlining this on Go 1.21.4 makes
				// BenchmarkUncontendedIteration 2.5 to 3 times slower.
				shift := 0
				for {
					// Reload the words every time because the user might have
					// set or unset bits we are yet to visit.
					w := ^word(0)
					for _, bs := range bss {
						w &= atomicLoadWord(&bs.words[j/wordBits])
						if w == 0 {
							continue WordLoop
						}
					}

					shift += trailingZerosWord(w >> shift)
					if shift >= wordBits {
						break
					}
					if !yield(j + shift) {
						return
					}
					shift++
				}
			}
		}
	}
}

/*
func AndCoarse(bss ...BitSet) iter.Seq2[int, int] {

}
*/

func divRoundUp(x, y int) int {
	return (x + y - 1) / y
}
