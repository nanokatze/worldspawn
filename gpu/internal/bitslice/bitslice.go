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

func (bs BitSlice) FindAndSet(i int) int {
	if i%64 != 0 {
		panic("not implemented")
	}

WordLoop:
	for ; i < bs.len; i += 64 {
		shift := 0
		for {
			mask := uint64(1) << shift

			old := atomic.OrUint64(&bs.words[i/64], mask)
			if old == ^uint64(0) {
				continue WordLoop
			}
			if old&mask == 0 {
				return i + shift
			}

			shift = bits.TrailingZeros64(^old)
			if i+shift >= bs.len {
				return -1
			}
		}
	}

	return -1
}

func divRoundUp(x, y int) int {
	return (x + y - 1) / y
}
