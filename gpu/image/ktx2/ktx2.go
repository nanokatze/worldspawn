package ktx2 // TODO: implement common handling in imageio like Go's core image.Decode does?

import (
	"encoding/binary"
	"io"
	"math"

	"worldspawn/gpu"
	"worldspawn/gpu/vk"
)

type fileHeader struct {
	Magic                      [12]byte
	VkFormat                   uint32
	TypeSize                   uint32
	Width, Height, Depth       uint32
	LayerCount                 uint32
	FaceCount                  uint32
	MipLevelCount              uint32
	SupercompressionScheme     uint32
	DataFormatDescriptor       section32
	KeyValueData               section32
	SupercompressionGlobalData section64
}

type section32 struct {
	Offset uint32
	Length uint32
}

type section64 struct {
	Offset uint64
	Length uint64
}

type mipHeader struct {
	Offset             uint64
	Length             uint64
	UncompressedLength uint64
}

// TODO: could we unify decoder with encoder?
type Decoder struct {
	r io.ReaderAt

	// don't just keep the entire header
	header fileHeader
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
	// TODO: do this garbage at NewDecoder time.

	dim := 1
	if d.header.Height > 0 {
		dim++
		if d.header.Depth > 0 {
			dim++
		}
	}

	extent := [3]int{
		int(d.header.Width),
		max(int(d.header.Height), 1),
		max(int(d.header.Depth), 1),
	}

	config := gpu.MakeImageConfig(vk.Format(d.header.VkFormat), extent[:])
	if d.header.FaceCount == 6 {
		config = config.AsCube()
	}

	return config.
		WithLayers(max(int(d.header.LayerCount), 1) * int(d.header.FaceCount)).
		WithMips(len(d.mips))
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

// TODO: the api should look something like what image/* decode libraries offer,
// but also have ways to get the raw offsets and stuff

// TODO: move this into gpu?
func enqueueReadAt(jq *gpu.JobQueue, r io.ReaderAt, p gpu.Slice[byte], off int64) {
	enqueueHostCall(jq, func() {
		if _, err := r.ReadAt(p.Value(), off); err != nil {
			// We don't really have any way to report read failures for now.
			panic(err)
		}
	})
}

// TODO: move into gpu?
// TODO: come up with a better name?
func enqueueHostCall(jq *gpu.JobQueue, f func()) {
	var wg1 gpu.WaitGroup
	wg1.Add(1)
	var wg2 gpu.WaitGroup
	wg2.Add(1)

	wg1.EnqueueDone(jq)
	go func() {
		wg1.Wait()
		f()
		wg2.Done()
	}()
	wg2.EnqueueWait(jq)
}
