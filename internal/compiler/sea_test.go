package compiler

import (
	"testing"
)

type testType struct{}

func (t testType) String() string { return "testtype" }

var opY = DefOp("Y", nil)
var opX = DefOp("X", nil)

func BenchmarkCreationOfValues(b *testing.B) {
	sea := NewSea()

	b.ReportAllocs()

	i := int64(0)
	for b.Loop() {
		sea.value(opY, testType{}, i)
		i++
	}
}

func BenchmarkGettingExistingValues(b *testing.B) {
	sea := NewSea()

	b.ReportAllocs()

	i := int64(0)
	for b.Loop() {
		sea.value(opY, testType{}, i)
	}
}

func TestStuff(t *testing.T) {
	testRules := []RewriteRule{
		{
			Pattern: &Pattern{
				Op:   opX,
				Args: []*Pattern{{}, {Op: opY}},
			},
			Rewrite: func(rr *RewriteResult, b *Builder, v *Value) {
				rr.Class(v.Arg(1))
			},
		},
	}

	sea := NewSea()
	b := &Builder{Sea: sea, Rules: testRules}

	c1 := b.Value2(opY, testType{}, 0)
	_ = b.Value2(opX, testType{}, nil, c1, c1)
	// _ = b.Value2(opX, testType{}, nil, c1, c1)

	// log.Println(c3.classes)

	Dump(sea, c1.Newest(), nil)
}

func BenchmarkBuilder(b *testing.B) {
	sea := NewSea()
	bld := &Builder{Sea: sea /*, RewriteRules: testRules*/}

	b.ReportAllocs()

	for b.Loop() {
		x := bld.Value2(opY, testType{}, 69)
		y := bld.Value2(opY, testType{}, 420)
		_ = bld.Value2(opX, testType{}, x, y)
	}
}
