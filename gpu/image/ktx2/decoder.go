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
	r          io.ReaderAt
	config     gpu.ImageConfig
	mipHeaders []mipSectionHeader
}

func (dec *Decoder) Reset(r io.ReaderAt) error {
	var header fileHeader
	if err := readStruct(r, 0, &header); err != nil {
		return err
	}

	// TODO: validation

	/*
		formatDescriptor, err := readSection(r, header.DataFormatDescriptor)
		if err != nil {
			return err
		}

		keyValues, err := readSection(r, header.KeyValueData)
		if err != nil {
			return err
		}
	*/

	// TODO: more validation

	mipHeaders := make([]mipSectionHeader, max(header.MipCount, 1))
	if err := readStruct(r, 80, mipHeaders); err != nil {
		return err
	}

	tmp := [3]int{
		int(header.Extent[0]),
		int(header.Extent[1]),
		int(header.Extent[2]),
	}

	*dec = Decoder{}
	dec.r = r
	dec.config = gpu.MakeImageConfig(vk.Format(header.Format), tmp[:index(tmp[:], 0)]).
		SetIsCube(header.FaceCount == 6).
		SetMips(int(max(header.MipCount, 1))).
		SetLayers(int(max(header.LayerCount, 1) * header.FaceCount))
	dec.mipHeaders = mipHeaders
	return nil
}

func (dec *Decoder) Config() gpu.ImageConfig { return dec.config }

func (dec *Decoder) DecodeGranularity() []int {
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
		mip := img.SubImage(gpu.SliceMips{i, i + 1})
		var g gpu.JobQueue
		mip.EnqueueInit(&g)
		dec.Decode(&g, mip, nil, i, nil, mip.Extent())
		g.Cleanup(mip.Destroy)
		wg.EnqueueDone(&g)
	}
	wg.Wait()

	return img, nil
}

func readStruct(r io.ReaderAt, off int64, p any) error {
	return binary.Read(io.NewSectionReader(r, off, math.MaxInt64), binary.LittleEndian, p)
}

func readSection(r io.ReaderAt, section sectionHeader32) ([]byte, error) {
	if section.Length == 0 {
		return nil, nil
	}

	buf := make([]byte, section.Length)
	n, err := r.ReadAt(buf, int64(section.Offset))
	return buf[:n], err
}
