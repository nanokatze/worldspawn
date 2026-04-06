package pathtracer

import (
	"worldspawn/gpu"
)

type AttributeBuffer struct {
	format uint32 // TODO: replace with an enum
	data   gpu.Pointer[byte]
	len    int
	stride int // must be multiple of format size and < 2^20
}
