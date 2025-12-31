package ecs

import (
	"iter"

	"worldspawn/internal/ecs/internal/bitset"
)

type Tuple[V1, V2 any] struct {
	V1 V1
	V2 V2
}

type Tuple3[V1, V2, V3 any] struct {
	V1 V1
	V2 V2
	V3 V3
}

// TODO: see if probing is worth it

// TODO: generate these using a template?

// TODO: right now we can end up in a situation when an ID was deleted from IDs,
// but not from components. In such case Get will (correctly) not return
// anything, but iterators (including Joins) will. Should we handle such cases
// by validating gens during iteration?

func Join[V1, V2 any](m1 Column[V1], m2 Column[V2]) iter.Seq2[ID, Tuple[V1, V2]] {
	return func(yield func(k ID, v Tuple[V1, V2]) bool) {
		ents := m1.ids
		bitset.And(m1.valid, m2.valid)(func(i int) bool {
			return yield(MakeID(i, ents.gens[i]), Tuple[V1, V2]{m1.data[i], m2.data[i]})
		})
	}
}

func Join3[V1, V2, V3 any](m1 Column[V1], m2 Column[V2], m3 Column[V3]) iter.Seq2[ID, Tuple3[V1, V2, V3]] {
	return func(yield func(k ID, v Tuple3[V1, V2, V3]) bool) {
		ents := m1.ids
		bitset.And(m1.valid, m2.valid, m3.valid)(func(i int) bool {
			return yield(MakeID(i, ents.gens[i]), Tuple3[V1, V2, V3]{m1.data[i], m2.data[i], m3.data[i]})
		})
	}
}
