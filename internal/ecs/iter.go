package ecs

import (
	"iter"

	"worldspawn/internal/ecs/bitslice"
)

// TODO: see if probing is worth it

// TODO: generate these using a template?

// TODO: make Column an interface and make iterators work over that?

// TODO: rename to Query, Query2, Query3, etc?

func All[T any](c *Column[T]) iter.Seq2[ID, T] {
	return func(yield func(k ID, v T) bool) {
		bitslice.And(c.valid)(func(i int) bool {
			return yield(MakeID(i, c.ids.gens[i]), c.data[i])
		})
	}
}

func Join[T0, T1 any](c0 *Column[T0], c1 *Column[T1]) iter.Seq2[ID, Tuple[T0, T1]] {
	return func(yield func(k ID, v Tuple[T0, T1]) bool) {
		rows := c0.ids

		bitslice.And(c0.valid, c1.valid)(func(i int) bool {
			return yield(MakeID(i, rows.gens[i]), Tuple[T0, T1]{c0.data[i], c1.data[i]})
		})
	}
}

func Join3[T0, T1, T2 any](c0 *Column[T0], c1 *Column[T1], c2 *Column[T2]) iter.Seq2[ID, Tuple3[T0, T1, T2]] {
	return func(yield func(k ID, v Tuple3[T0, T1, T2]) bool) {
		rows := c0.ids

		bitslice.And(c0.valid, c1.valid, c2.valid)(func(i int) bool {
			return yield(MakeID(i, rows.gens[i]), Tuple3[T0, T1, T2]{c0.data[i], c1.data[i], c2.data[i]})
		})
	}
}
