package ecs

import (
	"fmt"
	"iter"
	"reflect"

	"worldspawn/ecs/bitset"
)

// TODO: idk if I like the name "Reflect" or "Any". We'll want to think about it
// a bit more.
// TODO: move "Any" to be suffix?
type AnyComponentStore struct {
	vtyp    reflect.Type
	idAlloc *IDAlloc
	valid   bitset.Bitset
	data    reflect.Value
}

// TODO: make this again a method on ComponentStore?
func Reflect(v any) AnyComponentStore {
	cs := v.(interface{ anyComponentStore() AnyComponentStore })

	return cs.anyComponentStore()
}

func (m AnyComponentStore) ElemType() reflect.Type {
	return m.vtyp
}

func (m AnyComponentStore) All() iter.Seq[ID] {
	idAlloc := m.idAlloc

	return func(yield func(ID) bool) {
		for i := range bitset.And(m.valid) {
			if !yield(MakeID(i, idAlloc.gens[i])) {
				break
			}
		}
	}
}

func (m AnyComponentStore) Load(id ID, v reflect.Value) bool {
	// TODO: assert v.Type() == m.vtyp?
	// Prefabs will have some component stores be left uninitialized
	if m.idAlloc == nil {
		v.SetZero()
		return false
	}
	index := id.Index()
	if !m.idAlloc.Valid(id) || !m.valid.Test(index) {
		v.SetZero()
		return false
	}
	v.Set(m.data.Index(index))
	return true
}

func (m AnyComponentStore) Store(id ID, v reflect.Value) {
	// TODO: assert v.Type() == m.vtyp?
	if !m.idAlloc.Valid(id) {
		panic(fmt.Sprintf("bad %v", id))
	}
	index := id.Index()
	m.data.Index(index).Set(v)
	m.valid.Set(index)
}

func (m AnyComponentStore) Delete(id ID) {
	if !m.idAlloc.Valid(id) {
		panic("bad")
	}
	index := id.Index()
	if m.valid.Unset(index) {
		m.data.Index(index).SetZero() // don't retain pointers
	}
}

// TODO: should be suffixed to make it clear that it's for "AnyComponentStore"
func CopyComponentStore(dst, src AnyComponentStore) {
	// TODO: vtyp and other checks here please
	// TODO: checking idAlloc equivalence here would be expensive so we should
	// annotate the semantics of this function requiring idAlloc equivalence
	bitset.Copy(dst.valid, src.valid)
	reflect.Copy(dst.data, src.data)
}
