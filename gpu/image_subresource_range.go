package gpu

// TODO: use an even more compact representation
// TODO: this also needs an aspect/plane mask eventually
type subresourceRange struct {
	firstLayer uint32
	layers     uint32
	firstMip   uint8
	mips       uint8
}

func makeSubresourceRange(firstLayer, layers, firstMip, mips int) subresourceRange {
	return subresourceRange{
		firstLayer: uint32(firstLayer),
		layers:     uint32(layers),
		firstMip:   uint8(firstMip),
		mips:       uint8(mips),
	}
}

func (sr subresourceRange) FirstLayer() int { return int(sr.firstLayer) }

func (sr subresourceRange) Layers() int { return int(sr.layers) }

func (sr subresourceRange) FirstMip() int { return int(sr.firstMip) }

func (sr subresourceRange) Mips() int { return int(sr.mips) }
