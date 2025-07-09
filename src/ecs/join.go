package ecs

import (
	"iter"

	"worldspawn/ecs/bitset"
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

func Join[V1, V2 any](m1 ComponentStore[V1], m2 ComponentStore[V2]) iter.Seq2[ID, Tuple[V1, V2]] {
	idAlloc := m1.idAlloc

	return func(yield func(k ID, v Tuple[V1, V2]) bool) {
		bitset.And(m1.valid, m2.valid)(func(i int) bool {
			return yield(MakeID(i, idAlloc.gens[i]), Tuple[V1, V2]{m1.data[i], m2.data[i]})
		})
	}
}

func Join3[V1, V2, V3 any](m1 ComponentStore[V1], m2 ComponentStore[V2], m3 ComponentStore[V3]) iter.Seq2[ID, Tuple3[V1, V2, V3]] {
	idAlloc := m1.idAlloc

	return func(yield func(k ID, v Tuple3[V1, V2, V3]) bool) {
		bitset.And(m1.valid, m2.valid, m3.valid)(func(i int) bool {
			return yield(MakeID(i, idAlloc.gens[i]), Tuple3[V1, V2, V3]{m1.data[i], m2.data[i], m3.data[i]})
		})
	}
}
