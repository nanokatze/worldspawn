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

	header fileHeader // don't just keep the entire header around

	levelIndex []levelIndexEntry
}

func NewDecoder(r io.ReaderAt) (*Decoder, error) {
	var header fileHeader
	if err := binary.Read(io.NewSectionReader(r, 0, math.MaxInt64), binary.LittleEndian, &header); err != nil {
		return nil, err
	}

	levelIndex := make([]levelIndexEntry, max(header.LevelCount, 1))
	if err := binary.Read(io.NewSectionReader(r, 80, math.MaxInt64), binary.LittleEndian, levelIndex); err != nil {
		return nil, err
	}

	return &Decoder{
		r:          r,
		header:     header,
		levelIndex: levelIndex,
	}, nil
}

func (dec *Decoder) Config() gpu.ImageConfig {
	extent := [3]int{
		int(dec.header.Extent[0]),
		int(dec.header.Extent[1]),
		int(dec.header.Extent[2]),
	}

	dim := index(extent[:], 0)

	config := gpu.MakeImageConfig(vk.Format(dec.header.Format), extent[:dim]).
		AsCube(dec.header.FaceCount == 6).
		WithLayers(max(int(dec.header.LayerCount), 1) * int(dec.header.FaceCount)).
		WithMips(len(dec.levelIndex))
	return config
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
	dec, err := NewDecoder(r)
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
