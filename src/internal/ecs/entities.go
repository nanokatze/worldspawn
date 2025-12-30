package ecs

import (
	"fmt"

	"worldspawn/internal/ecs/internal/bitset"
)

// TODO: rename to Entities?
type Entities struct {
	used bitset.Bitset
	gens []uint32
	next int // wack; outsource hint management to the user
}

func NewEntities(n int) *Entities {
	return &Entities{
		used: bitset.Make(n),
		gens: make([]uint32, n),
		next: 1,
	}
}

// TODO: make this a standalone func
func (dst *Entities) Copy(src *Entities) {
	// TODO: ensure sizes match and stuff
	bitset.Copy(dst.used, src.used)
	copy(dst.gens, src.gens)
}

// TODO: kill off all alloc strategies and allocation onto the user

// TODO: bulk allocation?
// TODO: let the user control the ranges? the client needs to reserve IDs with
// high indices for client-only entities
// TODO: have this return error so it's up to the user to panic
func (a *Entities) Alloc() Entity {
	index := a.next
	for a.used.Set(index) {
		// BUG: this doesn't wrap around
		index++
	}
	a.next = index + 1
	gen := a.gens[index]
	id := MakeEntity(index, gen)
	if id == 0 {
		panic("unreachable")
	}
	return id
}

// TODO: rename
func (a *Entities) AllocAt(id Entity) error {
	// TODO: make sure this id is vacant
	index, gen := id.Index(), id.Generation()
	if a.used.Set(index) && a.gens[index] != gen {
		panic("bit was already set")
	}
	a.gens[index] = gen
	return nil
}

func (a *Entities) Free(id Entity) {
	index, gen := id.Index(), id.Generation()
	if tmp := a.gens[index]; tmp != gen {
		panic(fmt.Sprintf("a %v", id))
	}
	if !a.used.Unset(index) {
		panic(fmt.Sprintf("b %v", id))
	}
}

// TODO: rename to IDIsValid, etc?
func (a *Entities) Valid(id Entity) bool {
	index := id.Index()
	return a.used.Test(index) && a.gens[index] == id.Generation()
}

// TODO: rename?
func (a *Entities) Index(index int) Entity {
	if !a.used.Test(index) {
		return 0
	}
	return MakeEntity(index, a.gens[index])
}

func (a *Entities) Cap() int {
	return len(a.gens)
}
