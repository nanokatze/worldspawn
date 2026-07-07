package ecs

import "worldspawn/internal/ecs/bitset"

type IDs struct {
	used bitset.Bitset
	gens []uint32
}

func (ids *IDs) Cap() int {
	return len(ids.gens)
}

func (ids *IDs) Exists(id ID) bool {
	return ids.used.Test(id.Index()) && ids.gens[id.Index()] == id.Generation()
}

func (ids *IDs) Index(index int) ID {
	if !ids.used.Test(index) {
		return NullID
	}
	return MakeID(index, ids.gens[index])
}

/*
type IDConstraints struct {
	MinID ID

	MinIndex int
	MaxIndex int
}
*/

func (ids *IDs) nextFreeID(minIndex, maxIndex int, minID ID) ID {
	index := minID.Index()
	gen := minID.Generation()

	for range 2 {
		index = max(index, minIndex)
		if index == 0 && gen == 0 {
			index++
		}

		index = ids.nextFreeIndex(index)
		if index != -1 && index <= maxIndex {
			// TODO: assert minIndex <= index
			return MakeID(index, gen)
		}

		index = 0
		gen++
	}

	return NullID
}

func (ids *IDs) nextFreeIndex(i int) int {
	n := ids.Cap()
	// TODO: we should cook a FindFirstUnset on the bitset
	for ; i < n; i++ {
		if !ids.used.Test(i) {
			return i
		}
	}
	return -1
}

// TODO: add a constraint that the generation should advance? It seems like in
// the game we could run into a situation where an entity is revived (e.g.
// something unsets the deletion timer but that timer already fired on client.)
// We'll need to think that through I guess.
// TODO: what do we return if we try to create an ID that's already there?
func (ids *IDs) create(id ID) bool {
	if id == NullID {
		panic("null id")
	}
	index, gen := id.Index(), id.Generation()
	if ids.used.Set(index) {
		return false
	}
	ids.gens[index] = gen
	return true
}

func (ids *IDs) delete(id ID) {
	index := id.Index()
	if ids.gens[index] == id.Generation() {
		ids.used.Unset(index)
	}
}

func (dst *IDs) Copy(src *IDs) {
	// TODO: ensure sizes match

	bitset.Copy(dst.used, src.used)
	copy(dst.gens, src.gens)
}
