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

	config config

	levelIndex []levelIndexEntry
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

		config: config{
			dim:    dim,
			extent: header.Extent,
			format: vk.Format(header.Format),
			layers: max(header.LayerCount, 1) * header.FaceCount,
			mips:   max(header.LevelCount, 1),
		},

		levelIndex: levelIndex,
	}
	return nil
}

func (dec *Decoder) Config() *config { return &dec.config }

// TODO: support decoding at smaller granularity
// TODO: allow users to pass scratch data explicitly
func (dec *Decoder) Decode(
	jq *gpu.JobQueue,
	dst *gpu.Image, dstOffset []int,
	srcMip int, srcOffset []int,
	extent []int) {
	levelIndexEntry := &dec.levelIndex[srcMip]

	tmp := gpu.MakeSliceUncached[byte](int(levelIndexEntry.Length))
	defer jq.Cleanup(func() { gpu.Free(gpu.UnsafePointer(gpu.SliceData(tmp))) })

	enqueueReadAt(jq, dec.r, tmp, int64(levelIndexEntry.Offset))

	gpu.EnqueueCopyMemoryToImage(jq, dst, nil, tmp, 0, 0, extent)
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

		jq := new(gpu.JobQueue)
		level := img.SubImage(gpu.SliceMips{i, i + 1})
		level.EnqueueInit(jq)
		dec.Decode(jq, level, nil, i, nil, level.Extent())
		jq.Cleanup(level.Destroy)
		wg.EnqueueDone(jq)
	}
	wg.Wait()

	return img, nil
}
