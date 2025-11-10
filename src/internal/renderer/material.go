package renderer

import (
	"log"
	"math"
	"sync"

	"worldspawn/gpu"
	"worldspawn/internal/compiler"
	"worldspawn/internal/compiler/core"
	"worldspawn/internal/renderer/internal/material"
	"worldspawn/internal/renderer/matc"
)

/*
type MaterialSet struct {
	pipeline *gpu.RayTracingPipeline
	sbt      gpu.ShaderBindingTable
}
*/

// NOTE: This is almost like matc.InterpretedMaterial. I guess we should just
// make InterpretedMaterial be an interface with two implementations. We'll have
// to cook up InterpretedMaterial which will be basically
// matc.InterpretedMaterial but with code being gpu.Slice[uint32].
//
// TODO: make this an implementation of Material interface, and return the interface, and make this private I guess.
type InterpretedMaterial struct {
	program material.InterpreterProgram
	// TODO: other things like mapping string to offsets in the params bytes,
	// etc.
}

func (m *InterpretedMaterial) emissive() bool {
	return m.program.ABI.EDFCount > 0
}

func buildArith1(b *compiler.Builder, op compiler.Op, x *compiler.Class) *compiler.Class {
	return b.Value2(op, x.Type(), nil, x)
}

func buildArith2(b *compiler.Builder, op compiler.Op, x, y *compiler.Class) *compiler.Class {
	return b.Value2(op, x.Type(), nil, x, y)
}

func buildFract(b *compiler.Builder, v *compiler.Class) *compiler.Class {
	return buildArith2(b, core.OpFSub, v, buildArith1(b, core.OpFFloor, v))
}

func TestMaterial3(src []byte) *InterpretedMaterial {
	// TODO: do some json thing for now or parse some kind of subset of MtlX?
	return nil
}

var TestMaterial = sync.OnceValue(func() *InterpretedMaterial {
	sea := compiler.NewSea()
	b := &compiler.Builder{
		Sea:   sea,
		Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.LowerToInterpreter...),
	}

	normal := b.Value2(matc.OpInterpreterGetShadingNormal, core.MakeArrayType(3, core.Int32), nil)
	normal_x := core.ArrayExtract(b, normal, 0)
	normal_y := core.ArrayExtract(b, normal, 1)
	normal_z := core.ArrayExtract(b, normal, 2)

	uv := matc.LoadAttrGeometry(b, "UVs")
	u := core.ArrayExtract(b, uv, 0)
	v := core.ArrayExtract(b, uv, 1)

	one := core.Const(b, core.Int32, int64(math.Float32bits(1)))

	fractU := buildFract(b, u)
	fractV := buildFract(b, v)

	oneMinusFractU := buildArith2(b, core.OpFSub, one, fractU)
	oneMinusFractV := buildArith2(b, core.OpFSub, one, fractV)

	idk1 := buildArith2(b, core.OpFMin, fractU, fractV)
	idk2 := buildArith2(b, core.OpFMin, oneMinusFractU, oneMinusFractV)
	idk3 := buildArith2(b, core.OpFMin, idk1, idk2)

	selector := b.Value2(
		matc.OpInterpreterFLessOrEqualE8M23,
		core.Int32,
		nil,
		idk3,
		core.Const(b, core.Int32, int64(math.Float32bits(0.025))))

	color_r := core.CondSelect(
		b,
		one,
		matc.LoadAttrObject(b, "BaseColorR"),
		selector)
	color_g := core.CondSelect(
		b,
		one,
		matc.LoadAttrObject(b, "BaseColorG"),
		selector)
	color_b := core.CondSelect(
		b,
		one,
		matc.LoadAttrObject(b, "BaseColorB"),
		selector)

	program := b.Value2(
		matc.OpInterpreterPseudoOutput,
		core.MakeArrayType(6, core.Int32),
		&matc.InterpreterABI{
			BSDFs:   []material.BSDF{material.BSDFDiffuse},
			BSDFOff: 0,
		},
		color_r, color_g, color_b,
		normal_x, normal_y, normal_z)

	return NewMaterial(sea, program)
})

var TestMaterial2 = sync.OnceValue(func() *InterpretedMaterial {
	sea := compiler.NewSea()
	b := &compiler.Builder{
		Sea:   sea,
		Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.LowerToInterpreter...),
	}

	emission_r := core.Const(b, core.Int32, int64(math.Float32bits(10.0)))
	emission_g := core.Const(b, core.Int32, int64(math.Float32bits(10.0)))
	emission_b := core.Const(b, core.Int32, int64(math.Float32bits(10.0)))
	color := b.Value2(
		matc.OpInterpreterPseudoOutput,
		core.MakeArrayType(3, core.Int32),
		&matc.InterpreterABI{
			EDFs:   []material.EDF{material.EDFUniform},
			EDFOff: 0,
		},
		emission_r, emission_g, emission_b)

	return NewMaterial(sea, color)
})

// TODO: error material, which would be a pink/black emissive checkerboard

// TODO: we need a knob to specify whether to compile an interpreted material or
// an API shader material
func NewMaterial(sea *compiler.Sea, v *compiler.Class) *InterpretedMaterial {
	interpretedMaterial := matc.CompileInterpretedMaterial(sea, v)

	log.Println("outputs register", uint32(interpretedMaterial.Outputs))

	device := gpu.MakeSliceUncached[uint32](len(interpretedMaterial.Code))
	copy(device.Value(), interpretedMaterial.Code)

	var bsdfs [4]uint8
	for i, num := range interpretedMaterial.ABI.BSDFs {
		bsdfs[i] = uint8(num)
	}

	var edfs [1]uint8
	for i, num := range interpretedMaterial.ABI.EDFs {
		edfs[i] = uint8(num)
	}

	return &InterpretedMaterial{
		program: material.InterpreterProgram{
			ABI: material.InterpreterABI{
				BSDFs:     bsdfs,
				BSDFCount: uint8(len(interpretedMaterial.ABI.BSDFs)),
				BSDFsOff:  uint8(interpretedMaterial.ABI.BSDFOff),

				EDFs:     edfs,
				EDFCount: uint8(len(interpretedMaterial.ABI.EDFs)),
				EDFsOff:  uint8(interpretedMaterial.ABI.EDFOff),

				OutputsReg: uint32(interpretedMaterial.Outputs),
			},
			Code: gpu.SliceData(device),
		},
	}
}
