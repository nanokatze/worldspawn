package renderer

import (
	"math"
	"sync"
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/internal/renderer/internal/material"
)

type MaterialSet struct {
	pipeline *gpu.RayTracingPipeline
	sbt      gpu.ShaderBindingTable
}

// TODO: move this into material subpackage
// TODO: make types be prefixes rather than suffixes? idk
const (
	OpStop = iota

	OpMovk32

	OpAddF32
	OpSubF32
	OpMulF32
	OpDivF32
	OpMinF32
	OpMaxF32

	// TODO: get rid of in favor of x - floor(x)
	OpFracF32

	OpEqualF32
	OpNotEqualF32
	OpLessOrEqualF32

	OpConditionalSelect32

	OpLoad          // TODO: rename
	OpLoadAttribute // TODO: rename
	OpLoadNormal
)

type Material struct {
	// TODO: we also need a description of some sort where in the registers
	// inputs to different bxdfs lie, and the contributions of each bxdf.
	// TODO: if this material is emissive, we need an emissive-only program for
	// NEE.

	// TODO: rename to something like interpreterProgram
	code gpu.Slice[uint32]

	emissive gpu.Slice[uint32]
}

func packinstr(op, dst, src0, src1 int) uint32 {
	return uint32(op) | uint32(dst)<<8 | uint32(src0)<<16 | uint32(src1)<<24
}

var TestMaterial = sync.OnceValue(func() *Material {
	host := []uint32{
		packinstr(OpLoadAttribute, 0, 0, 0), uint32(unsafe.Offsetof(materialParams{}.UVs)),
		packinstr(OpFracF32, 0, 0, 0),
		packinstr(OpFracF32, 1, 1, 0),
		packinstr(OpMovk32, 6, 0, 0), math.Float32bits(1.0),
		packinstr(OpSubF32, 2, 6, 0),
		packinstr(OpSubF32, 3, 6, 1),
		packinstr(OpMinF32, 0, 0, 2),
		packinstr(OpMinF32, 1, 1, 3),
		packinstr(OpMinF32, 3, 0, 1),
		packinstr(OpMovk32, 4, 0, 0), math.Float32bits(0.01),
		packinstr(OpLessOrEqualF32, 5, 3, 4),
		packinstr(OpLoad, 0, 0, 0), uint32(unsafe.Offsetof(materialParams{}.BaseColor)),
		packinstr(OpLoad, 1, 0, 0), uint32(unsafe.Offsetof(materialParams{}.BaseColor) + 4),
		packinstr(OpLoad, 2, 0, 0), uint32(unsafe.Offsetof(materialParams{}.BaseColor) + 8),
		packinstr(OpConditionalSelect32, 3, 6, 0), 5,
		packinstr(OpConditionalSelect32, 4, 6, 1), 5,
		packinstr(OpConditionalSelect32, 5, 6, 2), 5,
		packinstr(OpLoadNormal, 0, 0, 0),
		// packinstr(OpMovk32, 0, 0, 0), math.Float32bits(1),
		packinstr(OpStop, 0, 0, 0),
	}

	device := gpu.MakeSliceUncached[uint32](len(host))
	copy(device.Value(), host)

	return &Material{
		code: device,
	}
})

var TestMaterial2 = sync.OnceValue(func() *Material {
	b := material.Builder{}
	_normal_x := b.Value(material.OpInterpreterMovk32, nil, nil, math.Float32bits(0.0))
	_normal_y := b.Value(material.OpInterpreterMovk32, nil, nil, math.Float32bits(0.0))
	_normal_z := b.Value(material.OpInterpreterMovk32, nil, nil, math.Float32bits(1.0))
	_diffuse_r := b.Value(material.OpInterpreterMovk32, nil, nil, math.Float32bits(0.0))
	_diffuse_g := b.Value(material.OpInterpreterMovk32, nil, nil, math.Float32bits(0.0))
	_diffuse_b := b.Value(material.OpInterpreterMovk32, nil, nil, math.Float32bits(0.0))
	_emission_r := b.Value(material.OpInterpreterMovk32, nil, nil, math.Float32bits(10.0))
	_emission_g := b.Value(material.OpInterpreterMovk32, nil, nil, math.Float32bits(3.33))
	_emission_b := b.Value(material.OpInterpreterMovk32, nil, nil, math.Float32bits(10.0))
	_color := b.Value(
		material.OpInterpreterPseudoMakeTuple,
		nil,
		[]*material.Value{
			_normal_x,
			_normal_y,
			_normal_z,
			_diffuse_r,
			_diffuse_g,
			_diffuse_b,
			_emission_r,
			_emission_g,
			_emission_b,
		},
		nil)

	tmp := NewMaterial(_color)
	tmp.emissive = tmp.code
	return tmp
})

func NewMaterial(v *material.Value) *Material {
	host := material.CompileInterpreterProgram(v)

	device := gpu.MakeSliceUncached[uint32](len(host))
	copy(device.Value(), host)

	return &Material{
		code: device,
	}
}
