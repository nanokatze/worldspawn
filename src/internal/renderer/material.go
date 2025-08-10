package renderer

import (
	"math"
	"sync"
	"unsafe"

	"worldspawn/gpu"
)

// TODO: move this into material subpackage
const (
	OpStop = iota

	OpMovk

	OpAddF32
	OpSubF32
	OpMulF32
	OpDivF32
	OpMinF32
	OpMaxF32

	OpFracF32

	OpEqualF32
	OpNotEqualF32
	OpLessOrEqualF32

	OpConditionalSelect

	OpLoad          // TODO: rename
	OpLoadAttribute // TODO: rename

	OpBSDFOrenNayarDiffuse
)

type Material struct {
	code gpu.Slice[uint32]
}

var TestMaterial = sync.OnceValue(func() *Material {
	host := []uint32{
		OpLoadAttribute, 0, uint32(unsafe.Offsetof(_MaterialParams{}.UVs)),
		OpFracF32, 0, 0,
		OpFracF32, 1, 1,
		OpMovk, 6, math.Float32bits(1.0),
		OpSubF32, 2, 6, 0,
		OpSubF32, 3, 6, 1,
		OpMinF32, 0, 0, 2,
		OpMinF32, 1, 1, 3,
		OpMinF32, 3, 0, 1,
		OpMovk, 4, math.Float32bits(0.01),
		OpLessOrEqualF32, 5, 3, 4,
		OpLoad, 0, uint32(unsafe.Offsetof(_MaterialParams{}.Hmm)),
		OpLoad, 1, uint32(unsafe.Offsetof(_MaterialParams{}.Hmm) + 4),
		OpLoad, 2, uint32(unsafe.Offsetof(_MaterialParams{}.Hmm) + 8),
		OpConditionalSelect, 0, 5, 6, 0,
		OpConditionalSelect, 1, 5, 6, 1,
		OpConditionalSelect, 2, 5, 6, 2,
		OpStop,
	}

	device := gpu.MakeSliceUncached[uint32](len(host))
	copy(device.Value(), host)

	return &Material{
		code: device,
	}
})
