package renderer

import (
	"log"
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
	code    gpu.Slice[uint32]
	outputs uint32 // start of the register array containing the outputs

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
		RewriteRules: material.LowerToMVM,
	}

	normal := b.Value2(
		material.OpMVMLoadNormal,
		compiler.MakeArrayType(compiler.Bits32, 3),
		nil)
	normal_x := compiler.BuildArrayExtract(&b, normal, 0)
	normal_y := compiler.BuildArrayExtract(&b, normal, 1)
	normal_z := compiler.BuildArrayExtract(&b, normal, 2)

	uv := b.Value2(
		material.OpMVMLoadAttribute,
		compiler.MakeArrayType(compiler.Bits32, 2),
		uint32(unsafe.Offsetof(materialParams{}.UVs)))
	u := compiler.BuildArrayExtract(&b, uv, 0)
	v := compiler.BuildArrayExtract(&b, uv, 1)

	one := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(1)))

	fractU := buildFract(&b, u)
	fractV := buildFract(&b, v)

	oneMinusFractU := buildArith2(&b, material.OpFSub, one, fractU)
	oneMinusFractV := buildArith2(&b, material.OpFSub, one, fractV)

	idk1 := buildArith2(&b, material.OpFMin, fractU, fractV)
	idk2 := buildArith2(&b, material.OpFMin, oneMinusFractU, oneMinusFractV)
	idk3 := buildArith2(&b, material.OpFMin, idk1, idk2)

	selector := b.Value2(
		material.OpMVMFLessOrEqualE8M23,
		compiler.Bits32,
		nil,
		idk3,
		compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0.025))))

	// TODO: use standard CondSelect and introduce a rule for scalarizing it and
	// introduce a rule to lower it to
	color_r := b.Value2(
		material.OpMVMConditionalSelect32,
		compiler.Bits32,
		nil,
		one,
		b.Value2(
			material.OpMVMLoad,
			compiler.Bits32,
			uint32(unsafe.Offsetof(materialParams{}.BaseColor))+0),
		selector)
	color_g := b.Value2(
		material.OpMVMConditionalSelect32,
		compiler.Bits32,
		nil,
		one,
		b.Value2(
			material.OpMVMLoad,
			compiler.Bits32,
			uint32(unsafe.Offsetof(materialParams{}.BaseColor))+4),
		selector)
	color_b := b.Value2(
		material.OpMVMConditionalSelect32,
		compiler.Bits32,
		nil,
		one,
		b.Value2(
			material.OpMVMLoad,
			compiler.Bits32,
			uint32(unsafe.Offsetof(materialParams{}.BaseColor))+8),
		selector)

	program := compiler.BuildMakeArray(
		&b,
		normal_x, normal_y, normal_z,
		color_r, color_g, color_b)

	return NewMaterial(sea, program)
})

var TestMaterial2 = sync.OnceValue(func() *Material {
	sea := compiler.NewSea()
	b := compiler.Builder{
		Sea:          sea,
		RewriteRules: material.LowerToMVM,
	}

	normal := b.Value2(
		material.OpMVMLoadNormal,
		compiler.MakeArrayType(compiler.Bits32, 3),
		nil)
	normal_x := compiler.BuildArrayExtract(&b, normal, 0)
	normal_y := compiler.BuildArrayExtract(&b, normal, 1)
	normal_z := compiler.BuildArrayExtract(&b, normal, 2)
	_diffuse_r := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0.0)))
	_diffuse_g := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0.0)))
	_diffuse_b := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(0.0)))
	_emission_r := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(10.0)))
	_emission_g := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(3.33)))
	_emission_b := compiler.BuildConst(&b, compiler.Bits32, int64(math.Float32bits(10.0)))
	_color := compiler.BuildMakeArray(&b,
		normal_x, normal_y, normal_z,
		_diffuse_r, _diffuse_g, _diffuse_b,
		_emission_r, _emission_g, _emission_b)

	tmp := NewMaterial(sea, _color)
	tmp.emissive = tmp.code
	return tmp
})

func NewMaterial(sea *compiler.Sea, v *compiler.Class) *Material {
	host, outputs := material.CompileMVMProgram(sea, v)

	device := gpu.MakeSliceUncached[uint32](len(host))
	copy(device.Value(), host)

	log.Println("outputs register", uint32(outputs))

	return &Material{
		code:    device,
		outputs: uint32(outputs),
	}
}
