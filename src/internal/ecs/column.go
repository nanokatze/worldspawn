package ecs

import (
	"errors"
	"iter"
	"reflect"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"

	"worldspawn/internal/ecs/internal/bitset"
)

type Column[V any] struct {
	ids   *IDs
	valid bitset.Bitset
	data  []V
}

// TODO: replace this somehow with doing something using AnyComponentStore?
// Actually there's no other way to autoinit stuff so we'll have to keep the
// Init method.
func (c *Column[V]) Init(ids *IDs) {
	c.ids = ids
	c.valid = bitset.Make(ids.Cap())
	c.data = make([]V, ids.Cap())
}

// Compatibility; TODO: remove, worldspawn code should implement save/restore
// and prefabs by itself
func (c *Column[V]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var tmp map[ID]V
	if err := json.UnmarshalDecode(dec, &tmp); err != nil {
		return err
	}
	{
		var maxIndex int
		for id := range tmp {
			maxIndex = max(maxIndex, id.Index())
		}
		if maxIndex >= len(c.data) {
			c.Init(NewEntities(maxIndex + 1))
		}
	}
	for id, v := range tmp {
		// if m.idAlloc
		if !c.ids.Exists(id) && !c.ids.Create(id) {
			return errors.New("bad")
		}
		c.Set(id, v)
	}
	return nil
}

func (c Column[V]) All() iter.Seq2[ID, V] {
	return func(yield func(k ID, v V) bool) {
		ents := c.ids
		bitset.And(c.valid)(func(i int) bool {
			return yield(MakeID(i, ents.gens[i]), c.data[i])
		})
	}
}

// TODO: there are many uses where Get is used without ok, should we make a separate method for that?
func (c Column[V]) Get(id ID) (V, bool) {
	// Prefabs will have some component stores be left uninitialized
	if c.ids == nil {
		return *new(V), false
	}
	index := id.Index()
	if !c.ids.Exists(id) || !c.valid.Test(index) {
		return *new(V), false
	}
	return c.data[index], true
}

func (c Column[V]) Set(id ID, v V) {
	index := id.Index()
	if !c.ids.Exists(id) {
		panic("bad")
	}
	c.data[index] = v
	c.valid.Set(index)
}

func (c Column[V]) Delete(id ID) {
	index := id.Index()
	if !c.ids.Exists(id) {
		panic("bad")
	}
	if c.valid.Unset(index) {
		c.data[index] = *new(V) // don't retain pointers
	}
}

func (c Column[V]) Clear() {
	// Don't retain pointers.
	//
	// Most ComponentStores the game calls Clear on are pretty sparse, so this
	// is faster than clearing the entire thing with a zeroing loop.
	//
	// TODO: we can peek inside hbitset to find non-zero counters and memzero
	// the huge range. This should provide decent performance regardless whether
	// ComponentStore is sparse or dense.
	for i := range bitset.And(c.valid) {
		c.data[i] = *new(V)
	}

	c.valid.Reset()
}

// TODO: make this a standalone function?
func (dst Column[V]) Copy(src Column[V]) {
	// TODO: we can do a better implementation, comparable to clear, which would
	// require peeking into the bitsets' counters or words. If we do not wish to
	// unnecessarily retain pointers, this would become a little bit more
	// complicated.
	// TODO: make sure the capacities are comparable
	bitset.Copy(dst.valid, src.valid)
	copy(dst.data, src.data)
}

type reflectedColumn[V any] struct{ column *Column[V] }

func (c *Column[V]) Reflect() ReflectedColumn { return reflectedColumn[V]{c} }

func (c reflectedColumn[V]) ElemType() reflect.Type { return reflect.TypeFor[V]() }

func (c reflectedColumn[V]) All() iter.Seq[ID] {
	column := *c.column
	return func(yield func(ID) bool) {
		ents := column.ids
		for i := range bitset.And(column.valid) {
			if !yield(MakeID(i, ents.gens[i])) {
				break
			}
		}
	}
}

func (c reflectedColumn[V]) Get(id ID, out reflect.Value) bool {
	v, ok := c.column.Get(id)
	*mustTypeAssert[*V](out.Addr()) = v
	return ok
}

func (c reflectedColumn[V]) Set(id ID, v reflect.Value) {
	c.column.Set(id, mustTypeAssert[V](v))
}

func (c reflectedColumn[V]) Delete(id ID) {
	c.column.Delete(id)
}

func (dst reflectedColumn[V]) Copy(src ReflectedColumn) {
	dst.column.Copy(*src.(reflectedColumn[V]).column)
}

func mustTypeAssert[T any](v reflect.Value) T {
	v2, ok := reflect.TypeAssert[T](v)
	if !ok {
		_ = v.Interface().(T)
	}
	return v2
}
