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
		sea.Value(OpConst, Int32, i)
		i++
	}
}

func BenchmarkGettingExistingValues(b *testing.B) {
	sea := NewSea()

	b.ReportAllocs()

	i := int64(0)
	for b.Loop() {
		sea.Value(OpConst, Int32, i)
	}
}

var testRules = []RewriteRule{
	{
		Pattern: &Pattern{
			Op:   OpArrayExtract,
			Args: []*Pattern{{Op: OpMakeArray, ArgsDDD: true}},
		},
		Rewrite: func(rc *RewriteContext, v *Value) {
			idx := v.Imm().(int64)
			for arr := range v.Args[0].Values2() {
				if arr.Op() == OpMakeArray {
					rc.Add3(arr.Args[idx])
				}
			}
		},
	},
	{
		Pattern: &Pattern{
			Op:   OpCondSelect,
			Args: []*Pattern{{}, {}, {}},
		},
		Rewrite: func(rc *RewriteContext, v *Value) {
			arr, ok := v.Type().(ArrayType)
			if !ok {
				return
			}

			x := v.Args[0]
			y := v.Args[1]
			cond := v.Args[2]

			elems := make([]*Class, arr.Len())
			for i := range arr.Len() {
				x_i := ArrayExtract(rc.B(), x, i)
				y_i := ArrayExtract(rc.B(), y, i)
				elems[i] = CondSelect(rc.B(), x_i, y_i, cond)
			}
			rc.Add2(OpMakeArray, arr, nil, elems...)
		},
	},
}

func BenchmarkBuilder(b *testing.B) {
	sea := NewSea()
	bld := &Builder{Sea: sea, RewriteRules: testRules}

	b.ReportAllocs()

	for b.Loop() {
		x := Const(bld, Int32, 69)
		y := Const(bld, Int32, 420)
		cond := bld.Value2(OpEqual, Int1, x, y)
		_ = CondSelect(bld, x, y, cond)
	}
}

// This actually scares me lmao
func TestGraphsWithCycles(t *testing.T) {
	sea := NewSea()

	b := Builder{Sea: sea}

	zero := b.Value2(OpConst, Int32, int64(0))
	x_of_zeros := b.Value2(opX, Int32, nil, zero, zero)

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
