package ecs

import (
	"fmt"

	"worldspawn/internal/ecs/bitset"
)

// TODO: rename into something else, e.g. IDManager
type IDAlloc struct {
	used bitset.Bitset
	gens []uint32
	next int // wack
}

func NewIDAlloc(n int) *IDAlloc {
	return &IDAlloc{
		used: bitset.Make(n),
		gens: make([]uint32, n),
		next: 1,
	}
}

// TODO: rename
func (a *IDAlloc) Reflect() (bitset.Bitset, []uint32) {
	return a.used, a.gens
}

// TODO: make this a standalone func
func (dst *IDAlloc) Copy(src *IDAlloc) {
	// TODO: ensure sizes match and stuff
	bitset.Copy(dst.used, src.used)
	copy(dst.gens, src.gens)
}

// TODO: bulk allocation?
// TODO: let the user control the ranges? the client needs to reserve IDs with
// high indices for client-only entities
// TODO: have this return error so it's up to the user to panic
func (a *IDAlloc) Alloc() ID {
	index := a.next
	for a.used.Set(index) {
		// BUG: this doesn't wrap around
		index++
	}
	a.next = index + 1
	gen := a.gens[index]
	id := MakeID(index, gen)
	if id == 0 {
		panic("unreachable")
	}
	return id
}

// TODO: rename
func (a *IDAlloc) AllocAt(id ID) error {
	// TODO: make sure this id is vacant
	index, gen := id.Index(), id.Generation()
	if a.used.Set(index) && a.gens[index] != gen {
		panic("bit was already set")
	}
	a.gens[index] = gen
	return nil
}

func (a *IDAlloc) Free(id ID) {
	index, gen := id.Index(), id.Generation()
	if tmp := a.gens[index]; tmp != gen {
		panic(fmt.Sprintf("a %v", id))
	}
	if !a.used.Unset(index) {
		panic(fmt.Sprintf("b %v", id))
	}
}

func (a *IDAlloc) Valid(id ID) bool {
	index := id.Index()
	return a.used.Test(index) && a.gens[index] == id.Generation()
}

func (a *IDAlloc) Index(index int) ID {
	if !a.used.Test(index) {
		return 0
	}
	return MakeID(index, a.gens[index])
}

// TODO: this is wack. Remove in favor of just Valid
func (a *IDAlloc) validate(id ID) (int, bool) {
	index := id.Index()
	if a.used.Test(index) && a.gens[index] == id.Generation() {
		return index, true
	}
	return -1, false
}

func (a *IDAlloc) Cap() int {
	return len(a.gens)
}
