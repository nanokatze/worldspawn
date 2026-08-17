package bitslice

import (
	"math/bits"
	"sync/atomic"
)

type BitSlice struct {
	words []uint64
	len   int
}

func Make(len int) BitSlice {
	return BitSlice{
		words: make([]uint64, divRoundUp(len, 64)),
		len:   len,
	}
}

func (bs BitSlice) Len() int { return bs.len }

func (bs BitSlice) Swap(i int, v bool) (previous bool) {
	// TODO: bounds checking

	mask := uint64(1) << (i % 64)

	var old uint64
	if v {
		old = atomic.OrUint64(&bs.words[i/64], mask)
	} else {
		old = atomic.AndUint64(&bs.words[i/64], ^mask)
	}
	return old&mask != 0
}

func (bs BitSlice) FindAndSet(start, n int) int {
	if start%64 != 0 {
		panic("not implemented")
	}

	for i := start; i < bs.len; i += 64 {
		for {
			word := atomic.LoadUint64(&bs.words[i/64])

			shift := findZeros(word, n)
			if shift == -1 {
				break
			}

			if atomic.CompareAndSwapUint64(&bs.words[i/64], word, word|(mask(n)<<shift)) {
				return i + shift
			}
		}
	}

	return -1
}

func mask(n int) uint64 {
	return uint64(1)<<n - 1
}

// findZeros finds a run of n zeros in w.
func findZeros(w uint64, n int) int {
	i := 0
	for {
		// Advance to the next zero
		i = i + bits.TrailingZeros64(^(w >> i))
		if i+n > 64 {
			return -1
		}

		// Find the next one
		j := i + bits.TrailingZeros64(w>>i)
		if i+n <= j {
			// We fit!
			return i
		}
		// i:j doesn't work for us, continue looking starting at i
		i = j
	}
}

func divRoundUp(x, y int) int {
	return (x + y - 1) / y
}
