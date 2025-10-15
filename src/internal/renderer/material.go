package renderer

import (
	"math"
	"sync"
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/internal/renderer/internal/compiler"
	"worldspawn/internal/renderer/internal/material"
)

type MaterialSet struct {
	pipeline *gpu.RayTracingPipeline
	sbt      gpu.ShaderBindingTable
}

type Material struct {
	// TODO: we also need a description of some sort where in the registers
	// inputs to different bxdfs lie, and the contributions of each bxdf.
	// TODO: if this material is emissive, we need an emissive-only program for
	// NEE.

	// TODO: rename to something like interpreterProgram
	code gpu.Slice[uint32]

	emissive gpu.Slice[uint32]
}

func buildArith1(b *compiler.Builder, op compiler.Op, x *compiler.Class) *compiler.Class {
	return b.Value2(op, x.Type(), nil, x)
}

func buildArith2(b *compiler.Builder, op compiler.Op, x, y *compiler.Class) *compiler.Class {
	return b.Value2(op, x.Type(), nil, x, y)
}

func buildFract(b *compiler.Builder, v *compiler.Class) *compiler.Class {
	floorv := buildArith1(b, material.OpFFloor, v)
	return buildArith2(b, material.OpFSub, v, floorv)
}

var TestMaterial = sync.OnceValue(func() *Material {
	sea := compiler.NewSea()
	b := compiler.Builder{
		Sea:          sea,
		RewriteRules: material.LowerToInterpreter,
	}

	normal := b.Value2(
		material.OpInterpreterLoadNormal,
		compiler.MakeTupleType(compiler.Bits32, compiler.Bits32, compiler.Bits32),
		nil)
	normal_x := compiler.BuildTupleExtract(&b, normal, 0)
	normal_y := compiler.BuildTupleExtract(&b, normal, 1)
	normal_z := compiler.BuildTupleExtract(&b, normal, 2)

	uv := b.Value2(
		material.OpInterpreterLoadAttribute,
		compiler.MakeTupleType(compiler.Bits32, compiler.Bits32),
		uint32(unsafe.Offsetof(materialParams{}.UVs)))

	u := compiler.BuildTupleExtract(&b, uv, 0)
	uFrac := buildFract(&b, u)
	v := compiler.BuildTupleExtract(&b, uv, 1)
	vFrac := buildFract(&b, v)

	idk := buildArith2(&b, material.OpFMin, uFrac, vFrac)

	selector := b.Value2(
		material.OpInterpreterFLessOrEqualE8M23,
		compiler.Bits32,
		nil,
		idk,
		compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0.1))))

	color := b.Value2(
		material.OpInterpreterConditionalSelect32,
		compiler.Bits32,
		nil,
		compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0))),
		compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(1))),
		selector)

	program := compiler.BuildMakeTuple(
		&b,
		normal_x, normal_y, normal_z,
		color, color, color)

	return NewMaterial(sea, program)
})

var TestMaterial2 = sync.OnceValue(func() *Material {
	sea := compiler.NewSea()
	b := compiler.Builder{
		Sea:          sea,
		RewriteRules: material.LowerToInterpreter,
	}

	normal := b.Value2(
		material.OpInterpreterLoadNormal,
		compiler.MakeTupleType(compiler.Bits32, compiler.Bits32, compiler.Bits32),
		nil)
	normal_x := compiler.BuildTupleExtract(&b, normal, 0)
	normal_y := compiler.BuildTupleExtract(&b, normal, 1)
	normal_z := compiler.BuildTupleExtract(&b, normal, 2)
	_diffuse_r := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0.0)))
	_diffuse_g := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0.0)))
	_diffuse_b := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0.0)))
	_emission_r := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(10.0)))
	_emission_g := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(3.33)))
	_emission_b := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(10.0)))
	_color := compiler.BuildMakeTuple(&b,
		normal_x, normal_y, normal_z,
		_diffuse_r, _diffuse_g, _diffuse_b,
		_emission_r, _emission_g, _emission_b)

	tmp := NewMaterial(sea, _color)
	tmp.emissive = tmp.code
	return tmp
})

func NewMaterial(sea *compiler.Sea, v *compiler.Class) *Material {
	host := material.CompileInterpreterProgram(sea, v)

	device := gpu.MakeSliceUncached[uint32](len(host))
	copy(device.Value(), host)

	return &Material{
		code: device,
	}
}
