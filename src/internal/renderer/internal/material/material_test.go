package material

import (
	"log"
	"math"
	"testing"

	"worldspawn/internal/renderer/internal/compiler"
)

func TestXxx(t *testing.T) {
	sea := compiler.NewSea()

	b := Builder{
		Sea:          sea,
		RewriteRules: LowerToInterpreter,
	}

	normal := b.Value2(
		OpInterpreterLoadNormal,
		compiler.MakeTupleType(compiler.Bits32, compiler.Bits32, compiler.Bits32),
		uint32(42))
	normal_x := compiler.BuildTupleExtract(&b, normal, 0)
	normal_y := compiler.BuildTupleExtract(&b, normal, 1)
	normal_z := compiler.BuildTupleExtract(&b, normal, 2)

	uv := b.Value2(OpInterpreterLoadAttribute, compiler.MakeTupleType(compiler.Bits32, compiler.Bits32), uint32(69))

	u := compiler.BuildTupleExtract(&b, uv, 0)
	v := compiler.BuildTupleExtract(&b, uv, 1)

	idk := buildArith2(&b, OpFMin, u, v)

	selector := b.Value2(
		OpInterpreterFLessOrEqualE8M23,
		compiler.Bits32,
		nil,
		idk,
		compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0.1))))

	color := b.Value2(
		OpInterpreterConditionalSelect32,
		compiler.Bits32,
		nil,
		compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0))),
		compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(1))),
		selector)

	program := compiler.BuildMakeTuple(
		&b,
		normal_x, normal_y, normal_z,
		color, color, color)

	compiledProgram := CompileInterpreterProgram(sea, program)
	for _, x := range compiledProgram {
		log.Printf("0x%08x", x)
	}
}
