package renderer

import (
	"math"
	"sync"
	"worldspawn/gpu"
)

// TODO: move this into material subpackage
const (
	OpStop = iota
	OpMovk
	OpLoad // TODO: rename to something else to make it clear that we're loading per-object data
	OpLoadAttribute
	OpFrac
	OpSub
	OpMin
	OpLessOrEqual
	OpCSel
)

type Material struct {
	// map of named params -> offsets. We'll also want to expand it with types
	// and stuff
	params map[string]int
	code   gpu.Slice[uint32]
}

var TestMaterial = sync.OnceValue(func() *Material {
	host := []uint32{
		OpLoadAttribute, 0,
		OpFrac, 0, 0,
		OpFrac, 1, 1,
		OpMovk, 6, math.Float32bits(1.0),
		OpSub, 2, 6, 0,
		OpSub, 3, 6, 1,
		OpMin, 0, 0, 2,
		OpMin, 1, 1, 3,
		OpMin, 3, 0, 1,
		OpMovk, 4, math.Float32bits(0.01),
		OpLessOrEqual, 5, 3, 4,
		OpLoad, 0, 28,
		OpLoad, 1, 32,
		OpLoad, 2, 36,
		OpCSel, 0, 5, 6, 0,
		OpCSel, 1, 5, 6, 1,
		OpCSel, 2, 5, 6, 2,
		OpStop,
	}

	device := gpu.MakeSliceUncached[uint32](len(host))
	copy(device.Value(), host)

	return &Material{
		params: map[string]int{
			"Color": 28,
		},
		code: device,
	}
})
