package ecs

import (
	"worldspawn/internal/ecs/internal/bitset"
)

type IDs struct {
	used bitset.Bitset
	gens []uint32
	next int // wack; outsource hint management to the user
}

func NewEntities(n int) *IDs {
	return &IDs{
		used: bitset.Make(n),
		gens: make([]uint32, n),
		next: 1,
	}
}

// TODO: make this a standalone func
func (dst *IDs) Copy(src *IDs) {
	// TODO: ensure sizes match and stuff
	bitset.Copy(dst.used, src.used)
	copy(dst.gens, src.gens)
}

// TODO: kill off all alloc strategies and allocation onto the user

// TODO: bulk allocation?
// TODO: let the user control the ranges? the client needs to reserve IDs with
// high indices for client-only entities
// TODO: have this return error so it's up to the user to panic
func (a *IDs) Alloc() ID {
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

func (a *IDs) Create(id ID) bool {
	index := id.Index()
	if index == 0 {
		panic("no")
	}
	if a.used.Set(index) {
		return false
	}
	a.gens[index] = id.Generation()
	return true
}

func (a *IDs) Delete(id ID) {
	index := id.Index()
	if a.gens[index] == id.Generation() {
		a.used.Unset(index)
	}
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

func (a *IDs) Cap() int {
	return len(a.gens)
}
