package compiler

import (
	"testing"
)

var opX = DefOp("X", nil)

func BenchmarkCreationOfValues(b *testing.B) {
	sea := NewSea()

	b.ReportAllocs()

	i := int64(0)
	for b.Loop() {
		sea.Value(OpConst, Bits32, i)
		i++
	}
}

// This actually scares me lmao
func TestGraphsWithCycles(t *testing.T) {
	sea := NewSea()

	b := Builder{Sea: sea}

	zero := b.Value2(OpConst, Bits32, int64(0))
	x_of_zeros := b.Value2(opX, Bits32, nil, zero, zero)

	class := equateClasses(sea, zero, x_of_zeros)

	Dump(sea, class, nil)
}

func equateClasses(sea *Sea, classes ...*Class) *Class {
	values := make(map[*Value]struct{})
	for _, c := range classes {
		for v := range c.Values {
			values[v] = struct{}{}
		}
	}
	return sea.EquateValues(values)
}
