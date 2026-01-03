package ecs

import "worldspawn/internal/ecs/internal/bitset"

type IDs struct {
	used bitset.Bitset
	gens []uint32
}

func (a *IDs) Cap() int {
	return len(a.gens)
}

// TODO: add a constraint that the generation should advance? It seems like in
// the game we could run into a situation where an entity is revived (e.g.
// something unsets the deletion timer but that timer already fired on client.)
// We'll need to think that through I guess.
// TODO: what do we return if we try to create an ID that's already there?
func (a *IDs) create(id ID) bool {
	if id == 0 {
		panic("null id")
	}
	index, gen := id.Index(), id.Generation()
	if a.used.Set(index) {
		return false
	}
	a.gens[index] = gen
	return true
}

func (a *IDs) Exists(id ID) bool {
	return a.used.Test(id.Index()) && a.gens[id.Index()] == id.Generation()
}

func (a *IDs) Index(index int) ID {
	if !a.used.Test(index) {
		return 0
	}
	return MakeID(index, a.gens[index])
}

func (dst *IDs) Copy(src *IDs) {
	// TODO: ensure sizes match

	bitset.Copy(dst.used, src.used)
	copy(dst.gens, src.gens)
}
