package renderer

import (
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

/*
func buildArith1(b *compiler.Builder, op compiler.Op, x *compiler.Class) *compiler.Class {
	return b.Value2(op, x.Type(), nil, x)
}

func buildArith2(b *compiler.Builder, op compiler.Op, x, y *compiler.Class) *compiler.Class {
	return b.Value2(op, x.Type(), nil, x, y)
}

func buildFract(b *compiler.Builder, v *compiler.Class) *compiler.Class {
	return buildArith2(b, core.OpFSub, v, buildArith1(b, core.OpFFloor, v))
}

var TestMaterial = sync.OnceValue(func() *InterpretedMaterial {
	sea := compiler.NewSea()
	b := &compiler.Builder{
		Sea:   sea,
		Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.LowerToInterpreter...),
	}

	normal := b.Value2(matc.OpInterpreterGetShadingNormal, core.ArrayType{3, core.Int32}, nil)

	uv := matc.LoadAttribute(b, "UVs")
	u := core.ArrayExtract(b, uv, 0)
	v := core.ArrayExtract(b, uv, 1)

	one := core.IConst(b, core.Int32, int64(math.Float32bits(1)))

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
		core.IConst(b, core.Int32, int64(math.Float32bits(0.025))))

	white := core.MakeArray(b, core.Int32, one, one, one)

	baseColor := core.MakeArray(b, core.Int32,
		matc.LoadObjectProperty(b, "BaseColorR"),
		matc.LoadObjectProperty(b, "BaseColorG"),
		matc.LoadObjectProperty(b, "BaseColorB"),
	)

	color := core.CondSelect(b, white, baseColor, selector)

	program := b.Value2(
		matc.OpMakeSurface,
		core.EmptyType{},
		nil,
		// bsdf,
		b.Value2(matc.OpDFComposition, matc.BSDFType{}, nil,
			color, b.Value2(matc.OpDiffuseBSDF, matc.BSDFType{}, nil, normal)),
		// edf
		b.Value2(matc.OpDFComposition, matc.EDFType{}, nil),
	)

	return NewMaterial(sea, program)
})
*/

var TestMaterial2 = sync.OnceValue(func() *InterpretedMaterial {
	sea := compiler.NewSea()
	b := &compiler.Builder{
		Sea:   sea,
		Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.LowerToInterpreter...),
	}

	emission_r := core.IConst(b, core.Int32, int64(math.Float32bits(1.0)))
	emission_g := core.IConst(b, core.Int32, int64(math.Float32bits(0)))
	emission_b := core.IConst(b, core.Int32, int64(math.Float32bits(1.0)))
	emissionSpectrum := core.MakeArray(b, core.Int32, emission_r, emission_g, emission_b)
	_ = emissionSpectrum
	program := b.Value2(
		matc.OpMakeSurface,
		core.EmptyType{},
		nil,
		// bsdf,
		b.Value2(matc.OpDFComposition, matc.BSDFType{}, nil),
		// edf
		b.Value2(matc.OpDFComposition, matc.EDFType{}, nil),

		// b.Value2(matc.OpDFComposition, matc.EDFType{}, nil,
		// 	emissionSpectrum, b.Value2(matc.OpUniformEDF, matc.EDFType{}, nil)),
	)

	return NewMaterial(sea, program)
})

// TODO: error material, which would be a pink/black emissive checkerboard

// TODO: we need a knob to specify whether to compile an interpreted material or
// an API shader material
func NewMaterial(sea *compiler.Sea, v *compiler.Class) *InterpretedMaterial {
	interpretedMaterial := matc.CompileInterpretedMaterial(sea, v)

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
