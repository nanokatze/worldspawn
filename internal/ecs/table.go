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

/*
type IDConstraints struct {
	MinIndex int
	MaxIndex int
}
*/

func (table *Table) CreateRowAuto(minIndex, maxIndex int, nextID *ID) ID {
	index := nextID.Index()
	gen := nextID.Generation()

	for range 2 {
		index = max(index, minIndex)
		if index == 0 && gen == 0 {
			index++
		}

		index = table.IDs().NextFreeIndex(index)
		if minIndex <= index && index <= maxIndex {
			id := MakeID(index, gen)
			if !table.CreateRow(id) {
				return NullID
			}
			*nextID = id.Succ()
			return id
		}

		index = 0
		gen++
	}

	return NullID
}

func (t *Table) CreateRow(id ID) bool {
	return t.IDs().create(id)
}

func (t *Table) DeleteRow(id ID) {
	for _, column := range t.columns {
		column.Delete(id)
	}
	t.IDs().delete(id)
}

// NOTE: this only copies IDs for now, but not columns.
func (dst *Table) Copy(src *Table) {
	// TODO: ensure sizes, etc, match.

	dst.IDs().Copy(src.IDs())

	// TODO: copy columns
}
