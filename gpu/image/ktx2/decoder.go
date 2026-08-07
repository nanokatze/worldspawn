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

	dim    uint8
	extent [3]uint32
	format vk.Format
	layers uint32

	levelIndex []levelIndexEntry
}

func NewDecoder(r io.ReaderAt) (*Decoder, error) {
	dec := new(Decoder)
	if err := dec.Reset(r); err != nil {
		return nil, err
	}
	return dec, nil
}

func (dec *Decoder) Reset(r io.ReaderAt) error {
	sr := io.NewSectionReader(r, 0, math.MaxInt64)

	var header fileHeader
	if err := binary.Read(sr, binary.LittleEndian, &header); err != nil {
		return err
	}

	// TODO: validation

	// TODO: reuse dec.levelIndex when possible
	levelIndex := make([]levelIndexEntry, max(header.LevelCount, 1))
	if err := binary.Read(sr, binary.LittleEndian, levelIndex); err != nil {
		return err
	}

	dim := uint8(index(header.Extent[:], 0))
	if header.FaceCount == 6 {
		dim |= 0x80
	}

	*dec = Decoder{
		r: r,

		dim:    dim,
		extent: header.Extent,
		format: vk.Format(header.Format),
		layers: max(header.LayerCount, 1) * header.FaceCount,

		levelIndex: levelIndex,
	}
	return nil
}

func (dec *Decoder) Config() gpu.ImageConfig {
	extent := [3]int{
		int(dec.extent[0]),
		int(dec.extent[1]),
		int(dec.extent[2]),
	}

	return gpu.MakeImageConfig(dec.format, extent[:dec.dim&0x7f]).
		AsCube(dec.dim&0x80 != 0).
		WithLayers(int(dec.layers)).
		WithMips(len(dec.levelIndex))
}

// TODO: support decoding at smaller granularity
// TODO: allow users to pass scratch data explicitly
func (dec *Decoder) EnqueueDecode(jq *gpu.JobQueue, dst *gpu.Image, mipIndex int) {
	levelEntry := dec.levelIndex[mipIndex]

	tmp := gpu.MakeSliceUncached[byte](int(levelEntry.Length))
	defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(gpu.SliceData(tmp))) })

	enqueueReadAt(jq, dec.r, tmp, int64(levelEntry.Offset))

	gpu.EnqueueCopyMemoryToImage(jq, dst, nil, tmp, 0, 0, dst.Extent())
}

func Decode(r io.ReaderAt, usage vk.ImageUsageFlags) (*gpu.Image, error) {
	var dec Decoder

	err := dec.Reset(r)
	if err != nil {
		return nil, err
	}

	config := dec.Config().WithUsage(vk.ImageUsageFlagBits(usage))

	img := gpu.NewImage(config)

	var wg gpu.WaitGroup
	for i := range config.Mips() {
		wg.Add(1)

		jq := new(gpu.JobQueue)
		mip := img.SubImage(gpu.SliceMips{i, i + 1})
		mip.EnqueueInit(jq)
		dec.EnqueueDecode(jq, mip, i)
		jq.Cleanup(mip.Destroy)
		wg.EnqueueDone(jq)
	}
	wg.Wait()

	return img, nil
}
