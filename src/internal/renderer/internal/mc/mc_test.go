package mc

import (
	"log"
	"math"
	"testing"
	"time"

	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/compiler/core"
)

var allRules = append(append([]compiler.RewriteRule(nil), core.Rules...), LowerToInterpreter...)

func TestXxx(t *testing.T) {
	t0 := time.Now()
	defer func() { log.Println("Create IR and Compile", time.Since(t0)) }()

	sea := compiler.NewSea()

	b := &Builder{
		Sea:   sea,
		Rules: allRules,
	}

	normal := b.Value(OpMVMLoadNormal, core.MakeArrayType(3, core.Int32))
	normal_x := core.ArrayExtract(b, normal, 0)
	normal_y := core.ArrayExtract(b, normal, 1)
	normal_z := core.ArrayExtract(b, normal, 2)

	uv := b.Value2(OpMVMLoadAttr, core.MakeArrayType(2, core.Int32), uint32(0x69))
	u := core.ArrayExtract(b, uv, 0)
	v := core.ArrayExtract(b, uv, 1)

	one := core.Const(b, core.Int32, int64(math.Float32bits(1)))

	fractU := u
	fractV := v

	oneMinusFractU := buildArith2(b, OpFSub, one, fractU)
	oneMinusFractV := buildArith2(b, OpFSub, one, fractV)

	idk1 := buildArith2(b, OpFMin, fractU, fractV)
	idk2 := buildArith2(b, OpFMin, oneMinusFractU, oneMinusFractV)
	idk3 := buildArith2(b, OpFMin, idk1, idk2)

	selector := b.Value2(
		OpMVMFLessOrEqualE8M23,
		core.Int32,
		nil,
		idk3,
		core.Const(b, core.Int32, int64(math.Float32bits(0.025))))

	color := core.MakeArray(
		b,
		core.Int32,
		b.Value2(
			OpMVMLoadParam,
			core.Int32,
			uint32(0x42)+0),
		b.Value2(
			OpMVMLoadParam,
			core.Int32,
			uint32(0x42)+4),
		b.Value2(
			OpMVMLoadParam,
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

	program := b.Value2(OpMVMPseudoOutput,
		core.MakeArrayType(6, core.Int32),
		&InterpretedMaterialOutputLayout{
			BSDFs:   []BSDF{BSDFDiffuse},
			BSDFOff: 0,
		},
		final_x, final_y, final_z,
		normal_x, normal_y, normal_z)

	compiled := Compile(sea, program, TargetInterpreter)

	t.Log(compiled.Outputs)
	t.Log(compiled.InterpretationTable)
}
