package ecs

import "worldspawn/internal/ecs/internal/bitset"

type IDs struct {
	used bitset.Bitset
	gens []uint32
}

func (ids *IDs) Cap() int {
	return len(ids.gens)
}

// TODO: add a constraint that the generation should advance? It seems like in
// the game we could run into a situation where an entity is revived (e.g.
// something unsets the deletion timer but that timer already fired on client.)
// We'll need to think that through I guess.
// TODO: what do we return if we try to create an ID that's already there?
func (ids *IDs) create(id ID) bool {
	if id == 0 {
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

func (ids *IDs) Exists(id ID) bool {
	return ids.used.Test(id.Index()) && ids.gens[id.Index()] == id.Generation()
}

func (ids *IDs) Index(index int) ID {
	if !ids.used.Test(index) {
		return 0
	}
	return MakeID(index, ids.gens[index])
}
