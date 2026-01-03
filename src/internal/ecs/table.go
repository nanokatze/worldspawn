package ecs

import (
	"worldspawn/internal/ecs/internal/bitset"
)

// TODO: factor out used and gens into its own object?
type Table struct {
	ids     IDs
	columns []ReflectedColumn
	next    int // wack; outsource hint management to the user
}

func NewTable(n int) *Table {
	return &Table{
		ids: IDs{
			used: bitset.Make(n),
			gens: make([]uint32, n),
		},
		next: 1,
	}
}

func (t *Table) IDs() *IDs {
	return &t.ids
}

// TODO: kill off all alloc strategies and allocation onto the user

// TODO: bulk allocation?
// TODO: let the user control the ranges? the client needs to reserve IDs with
// high indices for client-only entities
// TODO: have this return error so it's up to the user to panic
func (a *Table) Alloc() ID {
	index := a.next
	for a.ids.used.Set(index) {
		// BUG: this doesn't wrap around
		index++
	}
	a.next = index + 1
	gen := a.ids.gens[index]
	id := MakeID(index, gen)
	if id == 0 {
		panic("unreachable")
	}
	return id
}

func (t *Table) Create(id ID) bool {
	return t.ids.create(id)
}

func (a *IDs) delete(id ID) {
	index := id.Index()
	if a.gens[index] == id.Generation() {
		a.used.Unset(index)
	}
}

func (t *Table) Delete(id ID) {
	for _, column := range t.columns {
		column.Delete(id)
	}
	t.ids.delete(id)
}

// NOTE: this only copies IDs for now, but not columns.
func (dst *Table) Copy(src *Table) {
	// TODO: ensure sizes, etc, match.

	dst.ids.Copy(&src.ids)

	// TODO: copy columns
}
