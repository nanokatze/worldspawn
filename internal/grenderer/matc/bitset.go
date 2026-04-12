package matc

// TODO: make the bitset lazily allocated?

type bitset struct {
	words []uint64
}

func (bs bitset) Test(i int) bool {
	return bs.words[i/64]&(1<<(i%64)) != 0
}

func (bs bitset) Set(i int) {
	bs.words[i/64] |= 1 << (i % 64)
}

func (bs bitset) Unset(i int) {
	mask := uint64(1 << (i % 64))
	bs.words[i/64] &^= mask
}

func (bs bitset) FindAndSetMany(n int) int {
outer:
	for i := range 64*len(bs.words) - n {
		for j := range n {
			if bs.Test(i + j) {
				continue outer
			}
		}
		for j := range n {
			bs.Set(i + j)
		}
		return i
	}
	return -1
}

func (bs bitset) UnsetMany(i, n int) {
	for j := range n {
		bs.Unset(i + j)
	}
}
