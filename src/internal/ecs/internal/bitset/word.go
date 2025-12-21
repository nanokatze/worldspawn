package bitset

import (
	"math/bits"
	"sync/atomic"
)

type word uint64

func trailingZerosWord(x word) int {
	return bits.TrailingZeros64(uint64(x))
}

func atomicLoadWord(addr *word) word {
	return word(atomic.LoadUint64((*uint64)(addr)))
}

func atomicAndWord(addr *word, mask word) word {
	return word(atomic.AndUint64((*uint64)(addr), uint64(mask)))
}

func atomicOrWord(addr *word, mask word) word {
	return word(atomic.OrUint64((*uint64)(addr), uint64(mask)))
}
