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

func buildArith1(b compiler.Builder, op compiler.Op, x *compiler.Value) *compiler.Value {
	return b.Value(op, x.Type, []*compiler.Value{x}, nil)
}

func buildArith2(b compiler.Builder, op compiler.Op, x, y *compiler.Value) *compiler.Value {
	return b.Value(op, x.Type, []*compiler.Value{x, y}, nil)
}

func buildFract(b compiler.Builder, v *compiler.Value) *compiler.Value {
	floorv := buildArith1(b, material.OpInterpreterFFloor32, v)
	return buildArith2(b, material.OpInterpreterFSub32, v, floorv)
}

var TestMaterial = sync.OnceValue(func() *Material {
	sea := compiler.NewSea()
	b := material.Builder{
		Sea: sea,
	}

	normal := b.Value(
		material.OpInterpreterLoadNormal,
		compiler.MakeTupleType(compiler.Bits32, compiler.Bits32, compiler.Bits32),
		nil,
		uint32(unsafe.Offsetof(materialParams{}.UVs)))

	uv := b.Value(material.OpInterpreterLoadAttribute, compiler.MakeTupleType(compiler.Bits32, compiler.Bits32), nil, uint32(unsafe.Offsetof(materialParams{}.UVs)))

	u := b.Value(material.OpInterpreterPseudoTupleExtract, compiler.Bits32, []*compiler.Value{uv}, 0)
	uFrac := buildFract(b, u)
	v := b.Value(material.OpInterpreterPseudoTupleExtract, compiler.Bits32, []*compiler.Value{uv}, 1)
	vFrac := buildFract(b, v)

	idk := buildArith2(b, material.OpInterpreterFMin32, uFrac, vFrac)

	selector := b.Value(
		material.OpInterpreterFLessOrEqual32,
		compiler.Bits32,
		[]*compiler.Value{
			idk,
			b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(0.1)),
		},
		nil)

	color := b.Value(
		material.OpInterpreterConditionalSelect32,
		compiler.Bits32,
		[]*compiler.Value{
			b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(0)),
			b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(1)),
			selector,
		},
		nil)

	/*
		colorR := b.Value(material.OpInterpreterLoad, compiler.Bits32, nil, uint32(unsafe.Offsetof(materialParams{}.BaseColor))+0)
		colorG := b.Value(material.OpInterpreterLoad, compiler.Bits32, nil, uint32(unsafe.Offsetof(materialParams{}.BaseColor))+8)
		colorB := b.Value(material.OpInterpreterLoad, compiler.Bits32, nil, uint32(unsafe.Offsetof(materialParams{}.BaseColor))+12)
	*/

	program := compiler.BuildTuple(
		&b,
		material.OpInterpreterPseudoMakeTuple,
		normal,
		color, color, color)

	return NewMaterial(sea, program)
})

var TestMaterial2 = sync.OnceValue(func() *Material {
	sea := compiler.NewSea()
	b := material.Builder{
		Sea: sea,
	}

	_normal_x := b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(0.0))
	_normal_y := b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(0.0))
	_normal_z := b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(1.0))
	_diffuse_r := b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(0.0))
	_diffuse_g := b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(0.0))
	_diffuse_b := b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(0.0))
	_emission_r := b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(10.0))
	_emission_g := b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(3.33))
	_emission_b := b.Value(material.OpInterpreterConst32, compiler.Bits32, nil, math.Float32bits(10.0))
	_color := compiler.BuildTuple(&b,
		material.OpInterpreterPseudoMakeTuple,
		_normal_x, _normal_y, _normal_z,
		_diffuse_r, _diffuse_g, _diffuse_b,
		_emission_r, _emission_g, _emission_b)

	tmp := NewMaterial(sea, _color)
	tmp.emissive = tmp.code
	return tmp
})

func NewMaterial(sea *compiler.Sea, v *material.Value) *Material {
	host := material.CompileInterpreterProgram(sea, v)

	device := gpu.MakeSliceUncached[uint32](len(host))
	copy(device.Value(), host)

	return &Material{
		code: device,
	}
}
