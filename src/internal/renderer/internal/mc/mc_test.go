package mc

import (
	"math"
	"testing"

	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/compiler/core"
	"worldspawn/internal/renderer/internal/material"
)

var allRules = append(append([]compiler.RewriteRule(nil), core.Rules...), LowerToInterpreter...)

func TestXxx(t *testing.T) {
	sea := compiler.NewSea()

	b := &compiler.Builder{
		Sea:   sea,
		Rules: allRules,
	}

	normal := b.Value2(OpInterpreterLoadNormal, core.MakeArrayType(3, core.Int32), nil)
	normal_x := core.ArrayExtract(b, normal, 0)
	normal_y := core.ArrayExtract(b, normal, 1)
	normal_z := core.ArrayExtract(b, normal, 2)

	uv := b.Value2(OpInterpreterLoadAttr, core.MakeArrayType(2, core.Int32), uint32(0x69))
	u := core.ArrayExtract(b, uv, 0)
	v := core.ArrayExtract(b, uv, 1)

	one := core.Const(b, core.Int32, int64(math.Float32bits(1)))

	fractU := u
	fractV := v

	oneMinusFractU := buildArith2(b, core.OpFSub, one, fractU)
	oneMinusFractV := buildArith2(b, core.OpFSub, one, fractV)

	idk1 := buildArith2(b, core.OpFMin, fractU, fractV)
	idk2 := buildArith2(b, core.OpFMin, oneMinusFractU, oneMinusFractV)
	idk3 := buildArith2(b, core.OpFMin, idk1, idk2)

	selector := b.Value2(
		OpInterpreterFLessOrEqualE8M23,
		core.Int32,
		nil,
		idk3,
		core.Const(b, core.Int32, int64(math.Float32bits(0.025))))

	color := core.MakeArray(
		b,
		core.Int32,
		b.Value2(
			OpInterpreterLoadParam,
			core.Int32,
			uint32(0x42)+0),
		b.Value2(
			OpInterpreterLoadParam,
			core.Int32,
			uint32(0x42)+4),
		b.Value2(
			OpInterpreterLoadParam,
			core.Int32,
			uint32(0x42)+8))

	white := core.MakeArray(b, core.Int32, one, one, one)

	final := core.CondSelect(b, white, color, selector)
	final_x := core.ArrayExtract(b, final, 0)
	final_y := core.ArrayExtract(b, final, 1)
	final_z := core.ArrayExtract(b, final, 2)

	/*
		program := b.Value2(OpDFComposition, BSDFType{}, nil,
			final, b.Value2(OpDiffuseBSDF, BSDFType{}, nil, normal))
	*/

	program := b.Value2(OpInterpreterPseudoOutput,
		core.MakeArrayType(6, core.Int32),
		&material.InterpretedMaterialOutputLayout{
			BSDFs:   []material.BSDF{material.BSDFDiffuse},
			BSDFOff: 0,
		},
		final_x, final_y, final_z,
		normal_x, normal_y, normal_z)

	compiled := CompileForInterpreter(sea, program)

	t.Log(compiled.Outputs)
	t.Log(compiled.OutputLayout)
}
