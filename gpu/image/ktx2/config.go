package ktx2

import (
	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

type config struct {
	dim              uint8
	extent           [3]uint32
	format           vk.Format
	formatDescriptor []byte
	layers           uint32
	mips             uint32 // this could be an uint8
	kvd              map[string][]byte
}

func (c *config) Cube() bool { return c.dim&0x80 != 0 }

func (c *config) Extent() []int {
	tmp := [3]int{
		int(c.extent[0]),
		int(c.extent[1]),
		int(c.extent[2]),
	}
	return tmp[:c.dim&0x7f]
}

func (c *config) Mips() int {
	return int(c.mips)
}

func (c *config) gpuImageConfig() gpu.ImageConfig {
	return gpu.MakeImageConfig(c.format, c.Extent()).
		AsCube(c.Cube()).
		WithLayers(int(c.layers)).
		WithMips(c.Mips())
}
