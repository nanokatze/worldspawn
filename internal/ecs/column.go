package ecs

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"iter"
	"reflect"

	"worldspawn/internal/ecs/bitset"
)

type Column[T any] struct {
	table *Table // This is only necessary for json unmarshal garbage, kill
	ids   *IDs
	valid bitset.Bitset
	data  []T
}

// TODO: should we delegate registering the column with the table to the caller?
// In that case we will be able to replace *Table parameter with just *IDs.
func (c *Column[T]) Init(table *Table) {
	c.table = table
	c.ids = table.IDs()
	c.valid = bitset.Make(table.IDs().Cap())
	c.data = make([]T, table.IDs().Cap())
	table.columns = append(table.columns, c.Reflect())
}

func (c *Column[T]) Load(i int) T { return c.data[i] }

func (c *Column[T]) Store(i int, v T) {
	c.data[i] = v
	c.valid.Store(i, true)
}

func (c *Column[T]) Get(id ID) (T, bool) {
	// Prefabs will have some component stores be left uninitialized; TODO: remove
	if c.ids == nil {
		return *new(T), false
	}
	index := id.Index()
	if !c.ids.Exists(id) || !c.valid.Test(index) {
		return *new(T), false
	}
	return c.data[index], true
}

func (c *Column[T]) Set(id ID, v T) {
	index := id.Index()
	if !c.ids.Exists(id) {
		panic("bad")
	}
	c.data[index] = v
	c.valid.Set(index)
}

func (c *Column[T]) Delete(id ID) {
	index := id.Index()
	if !c.ids.Exists(id) {
		panic("bad")
	}
	c.data[index] = *new(T) // don't retain pointers
	c.valid.Unset(index)
}

func (dst *Column[T]) Copy(src *Column[T]) {
	// TODO: we can do a better implementation, comparable to clear, which would
	// require peeking into the bitsets' counters or words. If we do not wish to
	// unnecessarily retain pointers, this would become a little bit more
	// complicated.
	// TODO: make sure the capacities are comparable
	bitset.Copy(dst.valid, src.valid)
	copy(dst.data, src.data)
}

func (c *Column[T]) Clear() {
	// Don't retain pointers.
	//
	// Most ComponentStores the game calls Clear on are pretty sparse, so this
	// is faster than clearing the entire thing with a zeroing loop.
	//
	// TODO: we can peek inside hbitset to find non-zero counters and memzero
	// the huge range. This should provide decent performance regardless whether
	// ComponentStore is sparse or dense.
	for i := range bitset.And(c.valid) {
		c.data[i] = *new(T)
	}

	c.valid.Reset()
}

// Compatibility; TODO: remove, worldspawn code should implement save/restore
// and prefabs by itself
func (c *Column[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var tmp map[ID]T
	if err := json.UnmarshalDecode(dec, &tmp); err != nil {
		return err
	}
	{
		var maxIndex int
		for id := range tmp {
			maxIndex = max(maxIndex, id.Index())
		}
		if maxIndex >= len(c.data) {
			c.Init(NewTable(maxIndex + 1))
		}
	}
	for id, v := range tmp {
		// if m.idAlloc
		if !c.ids.Exists(id) && !c.table.CreateRow(id) {
			return errors.New("bad")
		}
		c.Set(id, v)
	}
	return nil
}

func (c *Column[T]) Reflect() ReflectedColumn { return (*reflectedColumn[T])(c) }

type reflectedColumn[T any] Column[T]

func (c *reflectedColumn[T]) ElemType() reflect.Type { return reflect.TypeFor[T]() }

func (c *reflectedColumn[T]) All() iter.Seq[ID] {
	return func(yield func(ID) bool) {
		for i := range bitset.And(c.valid) {
			if !yield(MakeID(i, c.ids.gens[i])) {
				break
			}
		}
	}
}

func (c *reflectedColumn[T]) Get(id ID, out reflect.Value) bool {
	v, ok := (*Column[T])(c).Get(id)
	*mustTypeAssert[*T](out.Addr()) = v
	return ok
}

func (c *reflectedColumn[T]) Set(id ID, v reflect.Value) {
	(*Column[T])(c).Set(id, mustTypeAssert[T](v))
}

func (c *reflectedColumn[T]) Delete(id ID) {
	(*Column[T])(c).Delete(id)
}

func (c *reflectedColumn[T]) Copy(src ReflectedColumn) {
	(*Column[T])(c).Copy((*Column[T])(src.(*reflectedColumn[T])))
}

func mustTypeAssert[T any](v reflect.Value) T {
	v2, ok := reflect.TypeAssert[T](v)
	if !ok {
		_ = v.Interface().(T)
	}
	return v2
}
