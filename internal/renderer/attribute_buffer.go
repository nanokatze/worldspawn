package renderer

import (
	"structs"

	"worldspawn/gpu"
)

type AttributeBuffer struct {
	_      structs.HostLayout
	format uint32 // TODO: replace with an enum
	data   gpu.Pointer[byte]
	len    int
	stride int // must be multiple of format size and < 2^20
}
