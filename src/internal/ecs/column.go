package ecs

import (
	"iter"
	"reflect"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"

	"worldspawn/internal/ecs/internal/bitset"
)

type Column[V any] struct {
	ents  *Entities
	valid bitset.Bitset
	data  []V
}

// TODO: replace this somehow with doing something using AnyComponentStore?
// Actually there's no other way to autoinit stuff so we'll have to keep the
// Init method.
func (c *Column[V]) Init(ents *Entities) {
	c.ents = ents
	c.valid = bitset.Make(ents.Cap())
	c.data = make([]V, ents.Cap())
}

// Compatibility; TODO: remove, worldspawn code should implement save/restore
// and prefabs by itself
func (c *Column[V]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var tmp map[Entity]V
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
		if err := c.ents.AllocAt(id); err != nil {
			return err
		}
		c.Set(id, v)
	}
	return nil
}

func (c Column[V]) All() iter.Seq2[Entity, V] {
	return func(yield func(k Entity, v V) bool) {
		ents := c.ents
		bitset.And(c.valid)(func(i int) bool {
			return yield(MakeEntity(i, ents.gens[i]), c.data[i])
		})
	}
}

// TODO: there are many uses where Get is used without ok, should we make a separate method for that?
func (c Column[V]) Get(id Entity) (V, bool) {
	// Prefabs will have some component stores be left uninitialized
	if c.ents == nil {
		return *new(V), false
	}
	index := id.Index()
	if !c.ents.Valid(id) || !c.valid.Test(index) {
		return *new(V), false
	}
	return c.data[index], true
}

func (c Column[V]) Set(id Entity, v V) {
	index := id.Index()
	if !c.ents.Valid(id) {
		panic("bad")
	}
	c.data[index] = v
	c.valid.Set(index)
}

func (c Column[V]) Delete(id Entity) {
	index := id.Index()
	if !c.ents.Valid(id) {
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

func (c reflectedColumn[V]) All() iter.Seq[Entity] {
	column := *c.column
	return func(yield func(Entity) bool) {
		ents := column.ents
		for i := range bitset.And(column.valid) {
			if !yield(MakeEntity(i, ents.gens[i])) {
				break
			}
		}
	}
}

func (c reflectedColumn[V]) Get(id Entity, v reflect.Value) bool {
	elem, ok := c.column.Get(id)
	if ok {
		v.Set(reflect.ValueOf(elem))
	} else {
		v.SetZero()
	}
	return ok
}

func (c reflectedColumn[V]) Set(id Entity, v reflect.Value) {
	c.column.Set(id, v.Interface().(V))
}

func (c reflectedColumn[V]) Delete(id Entity) {
	c.column.Delete(id)
}

func (dst reflectedColumn[V]) Copy(src ReflectedColumn) {
	dst.column.Copy(*src.(reflectedColumn[V]).column)
}
