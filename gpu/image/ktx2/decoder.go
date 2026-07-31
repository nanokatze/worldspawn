package ktx2

import (
	"encoding/binary"
	"io"
	"math"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

type Decoder struct {
	r io.ReaderAt

	header fileHeader // don't just keep the entire header around.
	mips   []mipHeader
}

func NewDecoder(r io.ReaderAt) (*Decoder, error) {
	var h fileHeader
	if err := binary.Read(io.NewSectionReader(r, 0, math.MaxInt64), binary.LittleEndian, &h); err != nil {
		return nil, err
	}

	mips := make([]mipHeader, max(h.MipLevelCount, 1))
	if err := binary.Read(io.NewSectionReader(r, 80, math.MaxInt64), binary.LittleEndian, mips); err != nil {
		return nil, err
	}

	return &Decoder{
		r:      r,
		header: h,
		mips:   mips,
	}, nil
}

func (d *Decoder) Config() gpu.ImageConfig {
	extent := [3]int{
		int(d.header.Extent[0]),
		int(d.header.Extent[1]),
		int(d.header.Extent[2]),
	}

	dim := index(extent[:], 0)

	config := gpu.MakeImageConfig(vk.Format(d.header.Format), extent[:dim]).
		AsCube(d.header.FaceCount == 6).
		WithLayers(max(int(d.header.LayerCount), 1) * int(d.header.FaceCount)).
		WithMips(len(d.mips))
	return config
}

// TODO: support decoding at smaller granularity
// TODO: allow users to pass scratch data explicitly
func (d *Decoder) EnqueueDecode(jq *gpu.JobQueue, dst *gpu.Image, mipIndex int) {
	mipHeader := d.mips[mipIndex]

	tmp := gpu.MakeSliceUncached[byte](int(mipHeader.Length))
	defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(gpu.SliceData(tmp))) })

	enqueueReadAt(jq, d.r, tmp, int64(mipHeader.Offset))

	gpu.EnqueueCopyMemoryToImage(jq, dst, nil, tmp, 0, 0, dst.Extent())
}
