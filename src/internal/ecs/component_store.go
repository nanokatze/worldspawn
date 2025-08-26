package ecs

import (
	"iter"
	"reflect"

	"github.com/go-json-experiment/json"
	"github.com/go-json-experiment/json/jsontext"

	"worldspawn/internal/ecs/bitset"
)

type ComponentStore[V any] struct {
	idAlloc *IDAlloc
	valid   bitset.Bitset
	data    []V
}

// TODO: replace this somehow with doing something using AnyComponentStore?
// Actually there's no other way to autoinit stuff so we'll have to keep the
// Init method.
func (m *ComponentStore[V]) Init(idAlloc *IDAlloc) {
	m.idAlloc = idAlloc
	m.valid = bitset.Make(idAlloc.Cap())
	m.data = make([]V, idAlloc.Cap())
}

// Compatibility; TODO: remove, worldspawn code should implement save/restore
// and prefabs by itself
func (m *ComponentStore[V]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	var tmp map[ID]V
	if err := json.UnmarshalDecode(dec, &tmp); err != nil {
		return err
	}
	{
		var maxIndex int
		for id := range tmp {
			maxIndex = max(maxIndex, id.Index())
		}
		if maxIndex >= len(m.data) {
			m.Init(NewIDAlloc(maxIndex + 1))
		}
	}
	for id, v := range tmp {
		// if m.idAlloc
		if err := m.idAlloc.AllocAt(id); err != nil {
			return err
		}
		m.Store(id, v)
	}
	return nil
}

func (m ComponentStore[V]) All() iter.Seq2[ID, V] {
	idAlloc := m.idAlloc

	return func(yield func(k ID, v V) bool) {
		bitset.And(m.valid)(func(i int) bool {
			return yield(MakeID(i, idAlloc.gens[i]), m.data[i])
		})
	}
}

// TODO: there are many uses where Load is used without ok, should we make a separate method for that?
func (m ComponentStore[V]) Load(id ID) (V, bool) {
	// Prefabs will have some component stores be left uninitialized
	if m.idAlloc == nil {
		return *new(V), false
	}
	index := id.Index()
	if !m.idAlloc.Valid(id) || !m.valid.Test(index) {
		return *new(V), false
	}
	return m.data[index], true
}

func (m ComponentStore[V]) Store(id ID, v V) {
	index := id.Index()
	if !m.idAlloc.Valid(id) {
		panic("bad")
	}
	m.data[index] = v
	m.valid.Set(index)
}

func (m ComponentStore[V]) Delete(id ID) {
	index := id.Index()
	if !m.idAlloc.Valid(id) {
		panic("bad")
	}
	if m.valid.Unset(index) {
		m.data[index] = *new(V) // don't retain pointers
	}
}

func (m ComponentStore[V]) Clear() {
	// Don't retain pointers.
	//
	// Most ComponentStores the game calls Clear on are pretty sparse, so this
	// is faster than clearing the entire thing with a zeroing loop.
	//
	// TODO: we can peek inside hbitset to find non-zero counters and memzero
	// the huge range. This should provide decent performance regardless whether
	// ComponentStore is sparse or dense.
	for i := range bitset.And(m.valid) {
		m.data[i] = *new(V)
	}

	m.valid.Reset()
}

// TODO: make this a standalone function?
func (dst ComponentStore[V]) Copy(src ComponentStore[V]) {
	// TODO: we can do a better implementation, comparable to clear, which would
	// require peeking into the bitsets' counters or words. If we do not wish to
	// unnecessarily retain pointers, this would become a little bit more
	// complicated.
	// TODO: make sure the capacities are comparable
	bitset.Copy(dst.valid, src.valid)
	copy(dst.data, src.data)
}

// TODO: this doesn't have any strong reason to be used by-pointer. Make this by-value?
// TODO: make this a standalone function?
func (m *ComponentStore[V]) anyComponentStore() AnyComponentStore {
	return AnyComponentStore{
		vtyp:    reflect.TypeFor[V](),
		idAlloc: m.idAlloc,
		valid:   m.valid,
		data:    reflect.ValueOf(m.data),
	}
}
