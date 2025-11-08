package renderer

import (
	"log"
	"math"
	"sync"

	"worldspawn/gpu"
	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/compiler/core"
	"worldspawn/internal/renderer/internal/matc"
	"worldspawn/internal/renderer/internal/material"
)

/*
type MaterialSet struct {
	pipeline *gpu.RayTracingPipeline
	sbt      gpu.ShaderBindingTable
}
*/

// This is almost like mc.InterpreterProgram. I guess we should just make
// Material be an interface with two implementations. We'll have to cook up
// InterpretedMaterial which will be basically mc.InterpreterProgram but with
// code being gpu.Slice[uint32].
type Material struct {
	code         gpu.Slice[uint32]
	outputsReg   int
	outputLayout material.InterpretedMaterialOutputLayout
	// TODO: other things like mapping string to offsets in the params bytes,
	// etc.
}

func (m *Material) emissive() bool {
	return len(m.outputLayout.EDFs) > 0
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

var TestMaterial = sync.OnceValue(func() *Material {
	sea := compiler.NewSea()
	b := compiler.Builder{
		Sea:   sea,
		Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.LowerToInterpreter...),
	}

	normal := b.Value2(matc.OpInterpreterGetShadingNormal, core.MakeArrayType(3, core.Int32), nil)
	normal_x := core.ArrayExtract(&b, normal, 0)
	normal_y := core.ArrayExtract(&b, normal, 1)
	normal_z := core.ArrayExtract(&b, normal, 2)

	uv := b.Value2(matc.OpInterpreterLoadAttrGeometry, core.MakeArrayType(2, core.Int32), "UVs")
	u := core.ArrayExtract(&b, uv, 0)
	v := core.ArrayExtract(&b, uv, 1)

	one := core.Const(&b, core.Int32, int64(math.Float32bits(1)))

	fractU := buildFract(&b, u)
	fractV := buildFract(&b, v)

	oneMinusFractU := buildArith2(&b, core.OpFSub, one, fractU)
	oneMinusFractV := buildArith2(&b, core.OpFSub, one, fractV)

	idk1 := buildArith2(&b, core.OpFMin, fractU, fractV)
	idk2 := buildArith2(&b, core.OpFMin, oneMinusFractU, oneMinusFractV)
	idk3 := buildArith2(&b, core.OpFMin, idk1, idk2)

	selector := b.Value2(
		matc.OpInterpreterFLessOrEqualE8M23,
		core.Int32,
		nil,
		idk3,
		core.Const(&b, core.Int32, int64(math.Float32bits(0.025))))

	// TODO: introduce a rule for scalarizing CondSelect. Note that right now we
	// fall over at extraction time, so...
	color_r := core.CondSelect(
		&b,
		one,
		b.Value2(matc.OpInterpreterLoadAttrObject, core.Int32, "BaseColorR"),
		selector)
	color_g := core.CondSelect(
		&b,
		one,
		b.Value2(matc.OpInterpreterLoadAttrObject, core.Int32, "BaseColorG"),
		selector)
	color_b := core.CondSelect(
		&b,
		one,
		b.Value2(matc.OpInterpreterLoadAttrObject, core.Int32, "BaseColorB"),
		selector)

	program := b.Value2(
		matc.OpInterpreterPseudoOutput,
		core.MakeArrayType(6, core.Int32),
		&material.InterpretedMaterialOutputLayout{
			BSDFs:   []material.BSDF{material.BSDFDiffuse},
			BSDFOff: 0,
		},
		color_r, color_g, color_b,
		normal_x, normal_y, normal_z)

	return newMaterial(sea, program)
})

var TestMaterial2 = sync.OnceValue(func() *Material {
	sea := compiler.NewSea()
	b := &compiler.Builder{
		Sea:   sea,
		Rules: append(append([]compiler.RewriteRule(nil), core.Rules...), matc.LowerToInterpreter...),
	}

	normal := b.Value2(matc.OpInterpreterGetShadingNormal, core.MakeArrayType(3, core.Int32), nil)
	normal_x := core.ArrayExtract(b, normal, 0)
	normal_y := core.ArrayExtract(b, normal, 1)
	normal_z := core.ArrayExtract(b, normal, 2)
	emission_r := core.Const(b, core.Int32, int64(math.Float32bits(10.0)))
	emission_g := core.Const(b, core.Int32, int64(math.Float32bits(10.0)))
	emission_b := core.Const(b, core.Int32, int64(math.Float32bits(10.0)))
	color := b.Value2(
		matc.OpInterpreterPseudoOutput,
		core.MakeArrayType(3, core.Int32),
		&material.InterpretedMaterialOutputLayout{
			EDFs:   []material.EDF{material.EDFUniform},
			EDFOff: 0,
		},
		emission_r, emission_g, emission_b,
		normal_x, normal_y, normal_z)

	return newMaterial(sea, color)
})

func newMaterial(sea *compiler.Sea, v *compiler.Class) *Material {
	interpreterProgram := matc.CompileForInterpreter(sea, v)

	device := gpu.MakeSliceUncached[uint32](len(interpreterProgram.Code))
	copy(device.Value(), interpreterProgram.Code)

	log.Println("outputs register", uint32(interpreterProgram.Outputs))

	return &Material{
		code:         device,
		outputsReg:   interpreterProgram.Outputs,
		outputLayout: interpreterProgram.OutputLayout,
	}
}
