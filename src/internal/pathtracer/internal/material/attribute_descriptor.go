package material

import (
	"math"
	"structs"
)

// TODO: make the internals private?
type AttributeDescriptor struct {
	_ structs.HostLayout
	// Domain uint32
	Data [4]uint32
}

func UniformAttribute(v [4]float32) AttributeDescriptor {
	return AttributeDescriptor{
		Data: float32x4bits(v),
	}
}

func float32x4bits(f [4]float32) [4]uint32 {
	return [4]uint32{
		math.Float32bits(f[0]),
		math.Float32bits(f[1]),
		math.Float32bits(f[2]),
		math.Float32bits(f[3]),
	}
}
