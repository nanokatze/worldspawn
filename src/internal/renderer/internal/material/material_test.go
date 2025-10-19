package material

import (
	"math"
	"testing"

	"worldspawn/internal/renderer/internal/compiler"
)

func TestXxx(t *testing.T) {
	sea := compiler.NewSea()

	b := Builder{
		Sea:          sea,
		RewriteRules: LowerToMVM,
	}

	normal := b.Value2(OpMVMLoadNormal, compiler.MakeArrayType(compiler.Bits32, 3), nil)
	normal_x := compiler.BuildArrayExtract(&b, normal, 0)
	normal_y := compiler.BuildArrayExtract(&b, normal, 1)
	normal_z := compiler.BuildArrayExtract(&b, normal, 2)

	uv := b.Value2(
		OpMVMLoadAttribute,
		compiler.MakeArrayType(compiler.Bits32, 2),
		uint32(69))

	u := compiler.BuildArrayExtract(&b, uv, 0)
	v := compiler.BuildArrayExtract(&b, uv, 1)

	idk := buildArith2(&b, OpFMin, u, v)

	selector := b.Value2(
		OpMVMFLessOrEqualE8M23,
		compiler.Bits32,
		nil,
		idk,
		compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0.1))))

	color := b.Value2(
		OpMVMConditionalSelect32,
		compiler.Bits32,
		nil,
		compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0))),
		compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(1))),
		selector)

	program := compiler.BuildMakeArray(
		&b,
		normal_x, normal_y, normal_z,
		color, color, color)

	_, outputs := CompileMVMProgram(sea, program)

	t.Log(outputs)
}
