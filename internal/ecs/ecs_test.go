package ecs

import "testing"

func TestRowCreation(t *testing.T) {
	rows := NewTable(100)

	// TODO: more row creation tests

	rows.CreateRow(MakeID(69, 1))
	rows.DeleteRow(MakeID(69, 1))

	rows.CreateRow(MakeID(69, 0))
	rows.DeleteRow(MakeID(69, 0))
}

func TestRowDeletion(t *testing.T) {
	rows := NewTable(100)

	var column Column[int]
	column.Init(rows)

	checkEmpty := func() {
		for i := 1; i < rows.IDs().Cap(); i++ {
			if _, ok := column.Get(MakeID(i, 0)); ok {
				t.Error("the column must be empty")
			}
		}
		for range All(&column) {
			t.Error("the column must be empty")
		}
	}

	id := MakeID(69, 0)

	rows.CreateRow(id)
	column.Set(id, 420)
	rows.DeleteRow(id)

	checkEmpty()

	rows.CreateRow(id)
	checkEmpty()
	rows.DeleteRow(id)
}
