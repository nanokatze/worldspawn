package ktx2 // TODO: implement common handling in gpu/image like Go's core image.Decode does?

import (
	"io"

	"worldspawn/gpu"
)

type fileHeader struct {
	Magic                      [12]byte
	Format                     uint32
	TypeSize                   uint32
	Extent                     [3]uint32
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
