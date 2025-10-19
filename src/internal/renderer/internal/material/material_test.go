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

	normal := b.Value2(
		OpMVMLoadNormal,
		compiler.MakeArrayType(compiler.Bits32, 3),
		nil)
	normal_x := compiler.BuildArrayExtract(&b, normal, 0)
	normal_y := compiler.BuildArrayExtract(&b, normal, 1)
	normal_z := compiler.BuildArrayExtract(&b, normal, 2)

	uv := b.Value2(
		OpMVMLoadAttribute,
		compiler.MakeArrayType(compiler.Bits32, 2),
		uint32(0x69))
	u := compiler.BuildArrayExtract(&b, uv, 0)
	v := compiler.BuildArrayExtract(&b, uv, 1)

	one := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(1)))

	fractU := u
	fractV := v

	oneMinusFractU := buildArith2(&b, OpFSub, one, fractU)
	oneMinusFractV := buildArith2(&b, OpFSub, one, fractV)

	idk1 := buildArith2(&b, OpFMin, fractU, fractV)
	idk2 := buildArith2(&b, OpFMin, oneMinusFractU, oneMinusFractV)
	idk3 := buildArith2(&b, OpFMin, idk1, idk2)

	selector := b.Value2(
		OpMVMFLessOrEqualE8M23,
		compiler.Bits32,
		nil,
		idk3,
		compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0.025))))

	// TODO: use standard CondSelect and introduce a rule for scalarizing it and
	// introduce a rule to lower it to
	color_r := b.Value2(
		OpMVMConditionalSelect32,
		compiler.Bits32,
		nil,
		one,
		b.Value2(
			OpMVMLoad,
			compiler.Bits32,
			uint32(0x42)+0),
		selector)
	color_g := b.Value2(
		OpMVMConditionalSelect32,
		compiler.Bits32,
		nil,
		one,
		b.Value2(
			OpMVMLoad,
			compiler.Bits32,
			uint32(0x42)+4),
		selector)
	color_b := b.Value2(
		OpMVMConditionalSelect32,
		compiler.Bits32,
		nil,
		one,
		b.Value2(
			OpMVMLoad,
			compiler.Bits32,
			uint32(0x42)+8),
		selector)

	program := compiler.BuildMakeArray(
		&b,
		normal_x, normal_y, normal_z,
		color_r, color_g, color_b)

	_, outputs := CompileMVMProgram(sea, program)

	t.Log(outputs)
}
