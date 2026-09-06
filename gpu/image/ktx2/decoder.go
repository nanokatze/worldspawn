package ktx2

import (
	"encoding/binary"
	"io"
	"math"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
	"worldspawn/gpu/vk/formatutil"
)

type Decoder struct {
	r io.ReaderAt

	config gpu.ImageConfig

	mipHeaders []mipSectionHeader
}

func (dec *Decoder) Reset(r io.ReaderAt) error {
	sr := io.NewSectionReader(r, 0, math.MaxInt64)

	var header fileHeader
	if err := binary.Read(sr, binary.LittleEndian, &header); err != nil {
		return err
	}

	// TODO: validation

	// TODO: reuse dec.mipHeaders when possible
	mipHeaders := make([]mipSectionHeader, max(header.MipCount, 1))
	if err := binary.Read(sr, binary.LittleEndian, mipHeaders); err != nil {
		return err
	}

	extentInt := [3]int{int(header.Extent[0]), int(header.Extent[1]), int(header.Extent[2])}

	*dec = Decoder{
		r: r,

		config: gpu.MakeImageConfig(vk.Format(header.Format), extentInt[:index(extentInt[:], 0)]).
			WithCube(header.FaceCount == 6).
			WithLayers(int(max(header.LayerCount, 1) * header.FaceCount)).
			WithMips(int(max(header.MipCount, 1))),

		mipHeaders: mipHeaders,
	}
	return nil
}

func (dec *Decoder) Config() gpu.ImageConfig { return dec.config }

func (dec *Decoder) Granularity() []int {
	config := dec.Config()
	dim := len(config.Extent())
	tmp := formatutil.Describe(config.Format()).BlockExtent
	return tmp[:dim]
}

// TODO: support decoding at smaller granularity
// TODO: allow users to pass scratch data explicitly
func (dec *Decoder) Decode(
	g *gpu.JobQueue,
	dst *gpu.Image, dstOffset []int,
	srcMip int, srcOffset []int,
	extent []int) {
	mipHeader := &dec.mipHeaders[srcMip]

	tmp := gpu.MakeSliceUncached[byte](int(mipHeader.Length))
	defer g.Cleanup(func() { gpu.Free(gpu.UnsafePointer(gpu.SliceData(tmp))) })

	enqueueReadAt(g, dec.r, tmp, int64(mipHeader.Offset))

	gpu.EnqueueCopyMemoryToImage(g, dst, nil, tmp, 0, 0, extent)
}

func Decode(r io.ReaderAt, opts ...gpu.NewImageOption) (*gpu.Image, error) {
	var dec Decoder

	err := dec.Reset(r)
	if err != nil {
		return nil, err
	}

	config := dec.Config()

	img := gpu.NewImage(config, opts...)

	var wg gpu.WaitGroup
	for i := range config.Mips() {
		wg.Add(1)

		var g gpu.JobQueue
		mip := img.SubImage(gpu.SliceMips{i, i + 1})
		mip.EnqueueInit(&g)
		dec.Decode(&g, mip, nil, i, nil, mip.Extent())
		g.Cleanup(mip.Destroy)
		wg.EnqueueDone(&g)
	}
	wg.Wait()

	return img, nil
}
