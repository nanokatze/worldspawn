package renderer

import (
	"math"
	"sync"
	"unsafe"

	"worldspawn/gpu"
	"worldspawn/internal/renderer/material"
)

type MaterialSet struct {
	pipeline *gpu.RayTracingPipeline
	sbt      gpu.ShaderBindingTable
}

/*
const (
	Op2AddF32 = iota
	Op2SubF32
	Op2MulF32
	Op2DivF32
	Op2MinF32
	Op2MaxF32

	Op2EqualF32
	Op2NotEqualF32
	Op2LessOrEqualF32

	Op2Pow
)

const (
	Op1Round
	Op1Trunc
	Op1Floor
	Op1Ceil

	Op1Sqrt

	Op1Sin
	Op1Cos
	Op1Tan
)
*/

// TODO: move this into material subpackage
const (
	OpStop = iota

	// TODO: rename to OpMovk32
	OpMovk

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

	// TODO: rename to OpConditionalSelect32
	OpConditionalSelect32

	OpLoad          // TODO: rename
	OpLoadAttribute // TODO: rename
)

type Material struct {
	// TODO: we also need a description of some sort where in the registers
	// inputs to different bxdfs lie, and the contributions of each bxdf.
	// TODO: if this material is emissive, we need an emissive-only program for
	// NEE.

	bxdfs []int

	// TODO: rename to something like interpreterProgram
	code gpu.Slice[uint32]

	// TODO: replace with emissive program
	emissive bool
}

func packinstr(op, dst, src0, src1 int) uint32 {
	return uint32(op) | uint32(dst)<<8 | uint32(src0)<<16 | uint32(src1)<<24
}

var TestMaterial = sync.OnceValue(func() *Material {
	host := []uint32{
		packinstr(OpLoadAttribute, 0, 0, 0), uint32(unsafe.Offsetof(_MaterialParams{}.UVs)),
		packinstr(OpFracF32, 0, 0, 0),
		packinstr(OpFracF32, 1, 1, 0),
		packinstr(OpMovk, 6, 0, 0), math.Float32bits(1.0),
		packinstr(OpSubF32, 2, 6, 0),
		packinstr(OpSubF32, 3, 6, 1),
		packinstr(OpMinF32, 0, 0, 2),
		packinstr(OpMinF32, 1, 1, 3),
		packinstr(OpMinF32, 3, 0, 1),
		packinstr(OpMovk, 4, 0, 0), math.Float32bits(0.01),
		packinstr(OpLessOrEqualF32, 5, 3, 4),
		packinstr(OpLoad, 0, 0, 0), uint32(unsafe.Offsetof(_MaterialParams{}.Hmm)),
		packinstr(OpLoad, 1, 0, 0), uint32(unsafe.Offsetof(_MaterialParams{}.Hmm) + 4),
		packinstr(OpLoad, 2, 0, 0), uint32(unsafe.Offsetof(_MaterialParams{}.Hmm) + 8),
		packinstr(OpConditionalSelect32, 0, 6, 0), 5,
		packinstr(OpConditionalSelect32, 1, 6, 1), 5,
		packinstr(OpConditionalSelect32, 2, 6, 2), 5,
		packinstr(OpStop, 0, 0, 0),
	}

	device := gpu.MakeSliceUncached[uint32](len(host))
	copy(device.Value(), host)

	return &Material{
		code: device,
	}
})

var TestMaterial2 = sync.OnceValue(func() *Material {
	base := TestMaterial()

	return &Material{
		code:     base.code,
		emissive: true,
	}
})

func NewMaterial(v *material.Value) *Material {
	return nil
}
