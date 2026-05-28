package ecs

import (
	"worldspawn/internal/ecs/internal/bitset"
)

// TODO: factor out used and gens into its own object?
type Table struct {
	ids     IDs
	columns []ReflectedColumn
}

func NewTable(n int) *Table {
	return &Table{
		ids: IDs{
			used: bitset.Make(n),
			gens: make([]uint32, n),
		},
	}
}

func (t *Table) IDs() *IDs { return &t.ids }

func (table *Table) CreateRowAuto(minIndex, maxIndex int, nextID *ID) ID {
	id := table.IDs().nextFreeID(minIndex, maxIndex, *nextID)
	if id != NullID && table.CreateRow(id) {
		*nextID = id
	}
	return id
}

func (t *Table) CreateRow(id ID) bool {
	return t.IDs().create(id)
}

func (t *Table) DeleteRow(id ID) {
	t.ClearRow(id)
	t.IDs().delete(id)
}

func (t *Table) ClearRow(id ID) {
	for _, column := range t.columns {
		column.Delete(id)
	}
}

// NOTE: this only copies IDs for now, but not columns.
func (dst *Table) Copy(src *Table) {
	// TODO: ensure sizes, etc, match.

	dst.IDs().Copy(src.IDs())

	// TODO: copy columns
}
